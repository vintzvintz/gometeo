package content

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"sync"
	"time"

	"gometeo/mfmap"
	"gometeo/mfmap/handlers"
	"gometeo/obs"
)

// ContentConf holds runtime configuration injected at construction time.
type ContentConf struct {
	DayMin  int
	DayMax  int
	CacheId string
	Obs     *obs.Registry // optional; nil disables observability
}

// Meteo is a http.Handler holding and serving live maps and pictos
type Meteo struct {
	conf   ContentConf
	maps   mapStore
	pictos pictoStore
	mux    meteoMux
}

// meteoMux is a mutex-protected, hot-swappable wrapper of a standard http.ServeMux
type meteoMux struct {
	serveMux *http.ServeMux
	mutex    sync.Mutex
}

// mapStore is the collection of donwloaded and parsed maps.
// Key is the original upstream path with a slash (i.e. france is "/", not "/france" ).
// Maps are published under path MfMap.Path() which may be different
type mapStore struct {
	store map[string]*mfmap.MfMap
	mutex sync.Mutex
}

// pictoStore is the collection of available pictos.
// Pictos are shared among all maps ( not a member of MfMap)
// Key is the name of the picto (ex : p1j, p4n, ...)
type pictoStore struct {
	store map[string][]byte
	mutex sync.Mutex
	obs   *obs.Registry
}

// New returns an empty Meteo struct
func New(conf ContentConf) *Meteo {
	return &Meteo{
		conf:   conf,
		maps:   mapStore{store: make(map[string]*mfmap.MfMap)},
		pictos: pictoStore{store: make(map[string][]byte)},
	}
}

func (mc *Meteo) Close() {
	slog.Info("MeteoContent closed")
}

// Obs returns the observability registry attached at construction, or nil.
func (mc *Meteo) Obs() *obs.Registry {
	return mc.conf.Obs
}

// Ready reports whether the content store has received at least one map.
// TODO check age of  last successfull request
func (mc *Meteo) Ready() bool {
	mc.maps.mutex.Lock()
	defer mc.maps.mutex.Unlock()
	return len(mc.maps.store) > 0
}

// Receive calls ReceiveMaps and ReceivePictos in parallel
// returns a 'done' channel to signal when both input channels
// are processed and closed
func (mc *Meteo) Receive(
	maps <-chan *mfmap.MfMap,
	pictos <-chan mfmap.Picto,
) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer func() {
			close(done)
			//log.Println("Meteo.receive() exit")
		}()

		//log.Println("Meteo.receive() start")

		// consume maps and pictos
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			<-mc.ReceiveMaps(maps)
			wg.Done()
		}()
		wg.Add(1)
		go func() {
			<-mc.ReceivePictos(pictos)
			wg.Done()
		}()
		// wait until both input channels are closed
		wg.Wait()
	}()
	return done
}

func (mc *Meteo) ReceiveMaps(ch <-chan *mfmap.MfMap) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		for m := range ch {
			mc.maps.update(m, mc.conf.DayMin, mc.conf.DayMax)
			mc.rebuildMux()
		}
	}()
	return done
}

func (mc *Meteo) ReceivePictos(ch <-chan mfmap.Picto) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		for p := range ch {
			mc.pictos.update(p)
			mc.rebuildMux()
		}
	}()
	return done
}

// pass request to MeteoContent internal ServeMux
func (mc *Meteo) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	// mux is replaced after Update() or Receive(), but this
	// implmentation detail is private to *content.Meteo
	mc.mux.ServeHTTP(resp, req)
}

func (mc *Meteo) Updatable() string {
	return mc.maps.updatable()
}

// MarkFailure records that a fetch attempt for the given upstream path failed,
// so the scheduler backs off before retrying. The path is matched against
// MfMap.OriginalPath, which is what Updatable() returns.
func (mc *Meteo) MarkFailure(originalPath string) {
	mc.maps.mutex.Lock()
	defer mc.maps.mutex.Unlock()
	for _, m := range mc.maps.store {
		if m.OriginalPath == originalPath {
			m.Schedule.MarkFailure()
			return
		}
	}
}

// StatusReport is the high-level data surfaced by the /statusse page.
// It combines the observability snapshot with per-store counts.
type StatusReport struct {
	Obs           obs.Snapshot
	MapsLoaded    int
	TotalHits     int64
	PictosLoaded  int
	NextUpdatable string
}

// Report returns a point-in-time status report. Safe to call concurrently.
func (mc *Meteo) Report() StatusReport {
	var snap obs.Snapshot
	if mc.conf.Obs != nil {
		snap = mc.conf.Obs.Snapshot()
	}
	mc.maps.mutex.Lock()
	mapsLoaded := len(mc.maps.store)
	var totalHits int64
	for _, m := range mc.maps.store {
		totalHits += m.Schedule.HitCount()
	}
	mc.maps.mutex.Unlock()

	mc.pictos.mutex.Lock()
	pictosLoaded := len(mc.pictos.store)
	mc.pictos.mutex.Unlock()

	return StatusReport{
		Obs:           snap,
		MapsLoaded:    mapsLoaded,
		TotalHits:     totalHits,
		PictosLoaded:  pictosLoaded,
		NextUpdatable: mc.Updatable(),
	}
}

func (mc *Meteo) rebuildMux() {
	newMux := http.NewServeMux()
	mc.pictos.register(newMux, mc.conf.CacheId, mc.conf.Obs)
	mc.maps.register(newMux, mc.conf.Obs)
	newMux.Handle("/statusse", mc.makeStatusHandler())
	mc.mux.setMux(newMux) // concurrent-safe accessor
}

// update()  adds or replace a map in the store.
// rebuilds all breadcrumbs in all maps in the store.
func (ms *mapStore) update(m *mfmap.MfMap, dayMin, dayMax int) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	path := m.Path()
	old, ok := ms.store[path]
	if ok {
		m.Merge(old, dayMin, dayMax)
	}
	ms.store[m.Path()] = m
	// TODO : optimize this quadractic algo
	for name := range ms.store {
		ms.buildBreadcrumbs(name)
	}
}

// Computes Breadcrumb chain for 'path' from other maps in the store
// NOT SAFE - ms.mutex must be acquired by callers.
func (ms *mapStore) buildBreadcrumbs(path string) {
	// get a *MfMap to work on
	m, ok := ms.store[path]
	if !ok || m == nil {
		slog.Warn("rebuildBreadcrumbs: map not found", "path", path)
		return // non fatal
	}

	// max depth is 3 France/Region/Dept
	bc := make(mfmap.Breadcrumbs, 0, 3)
	cur := m
	for {
		bc = append(bc, mfmap.BreadcrumbItem{
			Nom:  cur.Name(),
			Path: cur.Path(),
		})
		parent, ok := ms.store[cur.Parent]
		if !ok {
			break
		}
		cur = parent
	}
	slices.Reverse(bc)
	m.Breadcrumb = bc
}

func (ms *mapStore) register(mux *http.ServeMux, reg *obs.Registry) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	for _, m := range ms.store {
		handlers.Register(mux, m, reg)
	}
}

// returns map with the highest negative delay to update
func (ms *mapStore) updatable() (path string) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	var min time.Duration
	for _, m := range ms.store {
		d := m.Schedule.DurationToUpdate()
		if d <= min {
			min = d
			path = m.OriginalPath
		}
	}
	return
}

func (ps *pictoStore) update(p mfmap.Picto) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	ps.store[p.Name] = p.Img
}

func (ps *pictoStore) register(mux *http.ServeMux, cacheId string, reg *obs.Registry) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	ps.obs = reg
	pattern := fmt.Sprintf("/pictos/%s/{pic}", cacheId)
	mux.Handle(pattern, ps)
}

// ServeHTTP()
// last segment of the request URL /picto/cacheid/{pic} selects the picto to return
func (ps *pictoStore) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	name := req.PathValue("pic")
	b, ok := ps.store[name]
	if !ok {
		resp.WriteHeader(http.StatusNotFound)
		slog.Warn("picto not found", "name", name)
		return
	}
	resp.Header().Add("Cache-Control", "max-age=31536000, immutable")
	resp.Header().Add("Content-Type", "image/svg+xml")
	resp.WriteHeader(http.StatusOK)
	_, err := io.Copy(resp, bytes.NewReader(b))
	if err != nil {
		return
	}
	ps.obs.RecordPictoServed()
}

func (mux *meteoMux) setMux(newMux *http.ServeMux) {
	mux.mutex.Lock()
	defer mux.mutex.Unlock()
	mux.serveMux = newMux
}

func (mux *meteoMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mux.mutex.Lock()
	defer mux.mutex.Unlock()
	mux.serveMux.ServeHTTP(w, r)
}
