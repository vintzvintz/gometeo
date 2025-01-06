package crawl

import (
	"bytes"
	"io"
	"log"
	"net/http"

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
	for k, v := range maps {
		mc.maps[k] = v
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
			mc.maps[m.Path()] = m
			mc.rebuildMux()
		}
		log.Println("MeteoContent.Receive() exit")
		close(chDone)
	}()
	return chDone
}

func (mc *MeteoContent) rebuildMux() {
	mux := http.NewServeMux()
	mc.pictos.Register(mux)
	for _, m := range mc.maps {
		m.Register(mux)
	}
	mc.mux = mux
}

// pass request to MeteoContent internal ServeMux
func (mc *MeteoContent) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	mc.mux.ServeHTTP(resp, req)
}

// Register register itself to mux on "/" path
// should be registered last, after more specific paths like static assets
func (mc *MeteoContent) Register(mux *http.ServeMux) {
	mux.Handle("/", mc)
}

func (pictos pictoStore) Update(names []string, cr *Crawler) error {
	for _, name := range names {
		if _, ok := pictos[name]; ok {
			continue // do not update known pictos
		}
		b, err := cr.getPicto(name)
		if  err != nil {
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
