package content

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"slices"
	"sync"
	"time"

	"gometeo/mfmap"
)

// MeteoContent is a http.Handler holding and serving live maps and pictos
type Meteo struct {
	maps   mapStore
	pictos pictoStore
	mux    meteoMux
}

// Picto is a helper type for storage and passing data through channels
type Picto struct {
	Name string
	Img  []byte
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
}

// NewContent() returns an empty MeteoContent
func New() *Meteo {
	return &Meteo{
		maps:   mapStore{store: make(map[string]*mfmap.MfMap)},
		pictos: pictoStore{store: make(map[string][]byte)},
		// mux:    meteoMux{serveMux: &http.ServeMux{}},
	}
}

func (mc *Meteo) Close() {
	log.Println("MeteoContent closed")
}

// Receive calls ReceiveMaps and ReceivePictos in parallel
// returns a 'done' channel to signal when both input channels
// are processed and closed
func (mc *Meteo) Receive(
	maps <-chan *mfmap.MfMap,
	pictos <-chan Picto,
) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer func() {
			close(done)
			//log.Println("MeteoContent.receive() exit")
		}()

		//log.Println("MeteoContent.receive() start")

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
			mc.maps.update(m)
			mc.rebuildMux()
		}
	}()
	return done
}

func (mc *Meteo) ReceivePictos(ch <-chan Picto) <-chan struct{} {
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

func (mc *Meteo) rebuildMux() {
	newMux := http.NewServeMux()
	newMux.Handle("/pictos/{pic}", &mc.pictos)
	mc.maps.register(newMux)
	mc.mux.setMux(newMux) // concurrent-safe accessor
}

// update() adds or replace a map in the store.
// also updates breadcrumbs from other maps in the store
func (ms *mapStore) update(m *mfmap.MfMap) {
	// TODO: keep few days of past forecasts
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	path := m.Path()
	old := ms.store[path]
	merged := mfmap.Merge(old, m)
	ms.store[m.Path()] = merged
	// rebuild all breadcrumbs is not optimal
	for name := range ms.store {
		ms.buildBreadcrumbs(name)
	}
	//log.Printf("   mapStore.receive(%s)",m.Name())
}

// Computes Breadcrumb chain for 'path' from other maps in the store
// NOT SAFE - ms.mutex must be acquired by callers.
func (ms *mapStore) buildBreadcrumbs(path string) {
	// get a *MfMap to work on
	m, ok := ms.store[path]
	if !ok || m == nil {
		log.Printf("rebuildBreadcrumbs(): map '%s' not found", path)
		return // non fatal, abort without any modification
	}

	// max depth is 3 but lets allocate 5 to be sure
	bc := make(mfmap.Breadcrumb, 0, 5)
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

func (ms *mapStore) register(mux *http.ServeMux) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	for _, m := range ms.store {
		m.Register(mux)
	}
	mux.Handle("/statusse", ms.makeStatusHandler())
}

// returns map with the highest negative delay to update
func (ms *mapStore) updatable() (path string) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	var min time.Duration
	for _, m := range ms.store {
		d := m.DurationToUpdate()
		if d <= min {
			min = d
			path = m.OriginalPath
		}
	}
	return
}

func (ps *pictoStore) update(p Picto) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	ps.store[p.Name] = p.Img
}

// ServeHTTP()
// last segment of the request URL /picto/{pic} selects the picto to return
// the picto (svg picture) is written on resp as a []byte
func (ps *pictoStore) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	name := req.PathValue("pic")
	b, ok := ps.store[name]
	if !ok {
		resp.WriteHeader(http.StatusNotFound)
		log.Printf("error GET picto %s => statuscode%d\n", name, http.StatusNotFound)
		return
	}
	resp.Header().Add("Content-Type", "image/svg+xml")
	resp.WriteHeader(http.StatusOK)
	_, err := io.Copy(resp, bytes.NewReader(b))
	if err != nil {
		return
	}
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
