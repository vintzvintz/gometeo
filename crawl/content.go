package crawl

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"slices"

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
type mapStore map[string]*mfmap.MfMap

// PictoStore is the collection of available pictos.
// Pictos are shared among all maps ( not a member of MfMap)
// Key is the name of the picto (ex : p1j, p4n, ...)
type pictoStore map[string][]byte

// NewContent() returns an empty MeteoContent
func NewContent() *MeteoContent {
	m := MeteoContent{
		maps:   make(mapStore),
		pictos: make(pictoStore),
	}
	m.rebuildMux()
	return &m
}

// Merge and register maps and pictos into current content
// also replace internal ServeMux instance with new new handlers
func (mc *MeteoContent) Update(maps mapStore, pictos pictoStore) {
	for _, m := range maps {
		mc.maps.Add(m)
	}
	for k, v := range pictos {
		mc.pictos[k] = v
	}
	mc.rebuildMux()
}

// Receive updates continuously MeteoContent with MfMaps received from ch
// a Crawler cr is required to download pictos
func (mc *MeteoContent) Receive(chMaps <-chan *mfmap.MfMap, cr *Crawler) <-chan struct{} {
	chDone := make(chan struct{})
	go func() {
		log.Println("MeteoContent.Receive() start")
		for m := range chMaps {
			// get pictos
			err := mc.pictos.Update(m.PictoNames(), cr)
			if err != nil {
				// non fatal error
				log.Printf("error fetching pictos for map '%s': %s", m.Path(), err)
			}
			// update content with new map, pictos already updated (in-place just above)
			mc.maps.Add(m)
			mc.rebuildMux()
		}
		log.Println("MeteoContent.Receive() exit")
		close(chDone)
	}()
	return chDone
}

// pass request to MeteoContent internal ServeMux
func (mc *MeteoContent) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
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

func (ms mapStore) Register(mux *http.ServeMux) {
	for _, m := range ms {
		m.Register(mux)
	}
}

// Add() adds or replace a map in the store.
// Computes Breadcrumb chain from other maps in the store
// TODO: separate forecast updates and map insertion 
// in 2 functions (insertion is more expensive)
func (ms mapStore) Add(m *mfmap.MfMap) {
	// TODO: keep few days of past forecasts
	ms[m.Path()] = m
	// rebuild all breadcrumbs is not optimal
	for name := range ms {
		ms.buildBreadcrumbs(name)
	}
}

func (ms mapStore) buildBreadcrumbs(path string) {
	// get a *MfMap to work on
	m, ok := ms[path]
	if !ok || m == nil {
		log.Printf("rebuildBreadcrumbs(): map '%s' not found", path)
		return   // non fatal, abort without any modification
	}

	// max depth is 3 but lets allocate 5 to be sure
	bc := make(mfmap.Breadcrumb, 0, 5)
	cur := m
	for {
		bc = append(bc, mfmap.BreadcrumbItem{
			Nom:  cur.Name(),
			Path: cur.Path(),
		})
		parent, ok := ms[cur.Parent]
		if !ok {
			break
		}
		cur = parent
	}
	slices.Reverse(bc)
	m.Breadcrumb = bc
}

func (pictos pictoStore) Update(names []string, cr *Crawler) error {
	for _, name := range names {
		if _, ok := pictos[name]; ok {
			continue // download only new pictos
		}
		b, err := cr.getPicto(name)
		if err != nil {
			return err
		}
		pictos[name] = b
	}
	return nil
}

func (pictos pictoStore) Register(mux *http.ServeMux) {
	mux.HandleFunc("/pictos/{pic}", pictos.makePictosHandler())
}

// makePictosHandler() returns a handler serving pictos in PictoStore
// last segment of the request URL /picto/{pic} selects the picto to return
// the picto (svg picture) is written on resp as a []byte
func (pictos pictoStore) makePictosHandler() func(http.ResponseWriter, *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		name := req.PathValue("pic")
		_, ok := pictos[name]
		if !ok {
			resp.WriteHeader(http.StatusNotFound)
			log.Printf("error GET picto %s => statuscode%d\n", name, http.StatusNotFound)
			return
		}
		resp.Header().Add("Content-Type", "image/svg+xml")
		resp.WriteHeader(http.StatusOK)
		_, err := io.Copy(resp, bytes.NewReader(pictos[name]))
		if err != nil {
			return
		}
	}
}
