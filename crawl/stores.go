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
	maps   MapStore
	pictos PictoStore
	mux    *http.ServeMux
}

// MapStore is the collection of donwloaded and parsed maps.
// key is the original upstream path with a slash (i.e. france is "/", not "/france" )
// path exposed to clients is MfMap.Path()
type MapStore map[string]*mfmap.MfMap

// PictoStore is the collection of available pictos
// pictos are shared asset and not a member of MfMap
// key is the name of the picto (ex : p1j, p4n, ...)
type PictoStore map[string][]byte

// NewContent() returns an empty MeteoContent
func NewContent() *MeteoContent {
	m := MeteoContent{
		maps:   make(MapStore),
		pictos: make(PictoStore),
	}
	m.rebuildMux()
	return &m
}

// Merge and register maps and pictos into current content
// also replace internal ServeMux instance with new new handlers
func (mc *MeteoContent) Update(maps MapStore, pictos PictoStore) {
	for k, v := range maps {
		mc.maps[k] = v
	}
	for k, v := range pictos {
		mc.pictos[k] = v
	}
	mc.rebuildMux()
}

func (mc *MeteoContent) rebuildMux() {
	mux := http.NewServeMux()
	mc.pictos.Register(mux)
	for _, m := range mc.maps {
		m.Register(mux)
	}
	mc.mux = mux
}

/*
func (mc *MeteoContent) UpdateItem(item *CrawlItem) {
	mc.Update(item.maps, item.pictos)
}
*/
// call internal ServeMux
func (mc *MeteoContent) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	mc.mux.ServeHTTP(resp, req)
}

// Register register itself to mux on "/" path
// should be registered last, after more specific paths like static assets
func (c *MeteoContent) Register(mux *http.ServeMux) {
	mux.Handle("/", c)
}

func (ps PictoStore) Update(names []string, cl *MfClient) error {
	for _, pic := range names {
		if _, ok := ps[pic]; ok {
			continue // do not update known pictos
		}
		url, err := mfmap.PictoURL(pic)
		if err != nil {
			return err
		}
		body, err := cl.Get(url.String(), CacheDefault)
		if err != nil {
			return err
		}
		defer body.Close()
		b, err := io.ReadAll(body)
		if err != nil {
			return err
		}
		ps[pic] = b
	}
	return nil
}

func (ps PictoStore) Register(mux *http.ServeMux) {
	mux.HandleFunc("/pictos/{pic}", ps.makePictosHandler())
}

// makePictosHandler() returns a handler serving pictos in PictoStore
// last segment of the request URL /picto/{pic} selects the picto to return
// the picto (svg picture) is written on resp as a []byte
func (ps PictoStore) makePictosHandler() func(http.ResponseWriter, *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		pic := req.PathValue("pic")
		_, ok := ps[pic]
		if !ok {
			resp.WriteHeader(http.StatusNotFound)
			log.Printf("error GET picto %s => statuscode%d\n", pic, http.StatusNotFound)
			return
		}
		resp.Header().Add("Content-Type", "image/svg+xml")
		resp.WriteHeader(http.StatusOK)
		_, err := io.Copy(resp, bytes.NewReader(ps[pic]))
		if err != nil {
			return
		}
	}
}
