package crawl

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"slices"
	"sync"
	"unicode/utf8"

	"gometeo/mfmap"
)

// MeteoContent is a http.Handler holding and serving live maps and pictos
type MeteoContent struct {
	maps   mapStore
	pictos pictoStore
	mux    *http.ServeMux
}

// MapStore is the collection of donwloaded and parsed maps.
// Key is the original upstream path with a slash (i.e. france is "/", not "/france" ).
// Maps are published under path MfMap.Path() which may be different
type mapStore struct {
	store map[string]*mfmap.MfMap
	mutex sync.Mutex
}

// PictoStore is the collection of available pictos.
// Pictos are shared among all maps ( not a member of MfMap)
// Key is the name of the picto (ex : p1j, p4n, ...)
type pictoStore struct {
	store map[string][]byte
	mutex sync.Mutex
}

type Picto struct {
	name string
	img  []byte
}

// NewContent() returns an empty MeteoContent
func newContent() *MeteoContent {
	m := MeteoContent{
		maps: mapStore{
			store: make(map[string]*mfmap.MfMap),
			mutex: sync.Mutex{},
		},
		pictos: pictoStore{
			store: make(map[string][]byte),
			mutex: sync.Mutex{},
		},
	}
	m.rebuildMux()
	return &m
}

// Merge and register maps and pictos into current content
// also replace internal ServeMux instance with new new handlers
func (mc *MeteoContent) Import(maps map[string]*mfmap.MfMap, pictos map[string][]byte) {
	for _, m := range maps {
		mc.maps.receive(m)
	}
	for name, img := range pictos {
		mc.pictos.receive(&Picto{name: name, img: img})
	}
	mc.rebuildMux()
}

func (mc *MeteoContent) Close() {
	log.Println("MeteoContent closed")
}

func (mc *MeteoContent) receive(
	chMaps chan *mfmap.MfMap,
	chPictos chan *Picto,
) <-chan struct{} {
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		log.Println("MeteoContent.receive() start")
	loop:
		for {
			select {
			case m, ok := <-chMaps:
				if !ok {
					break loop
				}
				mc.maps.receive(m)
			case p, ok := <-chPictos:
				if !ok {
					break loop
				}
				mc.pictos.receive(p)
			}
			mc.rebuildMux()
		}
		log.Println("MeteoContent.Receive() exit")
	}()
	return doneCh
}

// pass request to MeteoContent internal ServeMux
func (mc *MeteoContent) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	// mux is replaced after Update() or Receive(), but this
	// implmentation detail is private to *MeteoContent
	mc.mux.ServeHTTP(resp, req)
}

// Register registers itself to mux on "/" path
// should be registered last, after more specific paths like static assets
func (mc *MeteoContent) Register(mux *http.ServeMux) {
	mux.Handle("/", mc)
}

func (mc *MeteoContent) rebuildMux() {
	mux := http.NewServeMux()
	mc.pictos.Register(mux)
	mc.maps.Register(mux)
	mc.mux = mux
}

func (ms *mapStore) Register(mux *http.ServeMux) {
	mux.Handle("/statusse", ms.makeStatusHandler())
	for _, m := range ms.store {
		m.Register(mux)
	}
}

// Add() adds or replace a map in the store.
// Computes Breadcrumb chain from other maps in the store
func (ms *mapStore) receive(m *mfmap.MfMap) {
	// TODO: keep few days of past forecasts
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	ms.store[m.Path()] = m
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

func (ms *mapStore) makeStatusHandler() http.HandlerFunc {
	return func(resp http.ResponseWriter, _ *http.Request) {
		ms.mutex.Lock()
		defer ms.mutex.Unlock()

		// sort keys by name for displaying maps in constant ordre
		var names = make([]string, 0, len(ms.store))
		var maxLen int
		for k := range ms.store {
			names = append(names, k)
			// also finds max length
			n := utf8.RuneCountInString(k)
			if n > maxLen {
				maxLen = n
			}
		}
		slices.Sort(names)
		b := &bytes.Buffer{}
		for _, name := range names {
			m := ms.store[name]
			b.WriteString(m.Stats().Format(maxLen))
		}
		io.Copy(resp, b)
	}
}

func (ps *pictoStore) receive(p *Picto) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	ps.store[p.name] = p.img
}

func (pictos *pictoStore) Register(mux *http.ServeMux) {
	mux.HandleFunc("/pictos/{pic}", pictos.makePictosHandler())
}

// makePictosHandler() returns a handler serving pictos in PictoStore
// last segment of the request URL /picto/{pic} selects the picto to return
// the picto (svg picture) is written on resp as a []byte
func (pictos *pictoStore) makePictosHandler() func(http.ResponseWriter, *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		pictos.mutex.Lock()
		defer pictos.mutex.Unlock()

		name := req.PathValue("pic")
		b, ok := pictos.store[name]
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
}
