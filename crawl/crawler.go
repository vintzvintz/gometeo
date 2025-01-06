package crawl

import (
	"bytes"
	"encoding/gob"
	"io"
	"log"
	"net/http"
	"os"

	"gometeo/mfmap"
)

const (
	httpsMeteofranceCom = "https://meteofrance.com"
	sessionCookie       = "mfsession"
)

type Crawler struct {
	mainClient *MfClient
}

// MeteoContent is a http.Handler holding and serving live maps and pictos
type MeteoContent struct {
	maps   MapStore
	pictos PictoStore
	mux    *http.ServeMux
}

// NewContent() returns an empty MeteoContent
func NewContent() *MeteoContent {
	m := MeteoContent{}
	m.Update(make(MapStore), make(PictoStore))
	return &m
}

// utility type to store a MeteoContent without the ServeMux
type meteoBlob struct {
	Maps   MapStore
	Pictos PictoStore
}

func LoadContent(cacheFile string) *MeteoContent {
	f, err := os.Open(cacheFile)
	if err != nil {
		log.Println(err)
		return nil
	}
	dec := gob.NewDecoder(f)
	blob := meteoBlob{}
	err = dec.Decode(&blob)
	if err != nil {
		log.Println(err)
		return nil
	}
	log.Printf("loaded maps & pictos from %s", cacheFile)
	mc := NewContent() // empty but non-nil
	mc.Update(blob.Maps, blob.Pictos)
	return mc
}

func StoreContent(cacheFile string, mc *MeteoContent) {
	f, err := os.Create(cacheFile)
	if err != nil {
		panic(err)
	}
	blob := meteoBlob{
		Maps:   mc.maps,
		Pictos: mc.pictos,
	}
	enc := gob.NewEncoder(f)
	err = enc.Encode(blob)
	if err != nil {
		panic(err)
	}
	log.Printf("content stored to %s", cacheFile)
}

// MapStore is the collection of donwloaded and parsed maps.
// key is the original upstream path with a slash (i.e. france is "/", not "/france" )
// path exposed to clients is MfMap.Path()
type MapStore map[string]*mfmap.MfMap

// PictoStore is the collection of available pictos
// pictos are shared asset and not a member of MfMap
// key is the name of the picto (ex : p1j, p4n, ...)
type PictoStore map[string][]byte

// NewCrawler allocates as *MfCrawler
func NewCrawler() *Crawler {
	return &Crawler{
		mainClient: NewClient(httpsMeteofranceCom),
	}
}

// GetMap gets https://mf.com/zone html page and related data like
// svg map, pictos, forecasts and list of subzones
// related data is stored into MfMap fields
func (c *Crawler) GetMap(path string, parent *mfmap.MfMap, pictos PictoStore) (*mfmap.MfMap, error) {
	log.Printf("GetMap() '%s'", path)

	body, err := c.mainClient.Get(path, CacheDisabled)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	// initialise map
	m := mfmap.MfMap{
		//		Nom: nom,
		Parent: parent,
	}
	err = m.ParseHtml(body)
	if err != nil {
		return nil, err
	}

	// get svg map
	u, err := m.SvgURL()
	if err != nil {
		return nil, err
	}
	body, err = c.mainClient.Get(u.String(), CacheDefault)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	err = m.ParseSvgMap(body)
	if err != nil {
		return nil, err
	}

	// get geography data
	u, err = m.GeographyURL()
	if err != nil {
		return nil, err
	}
	body, err = c.mainClient.Get(u.String(), CacheDefault)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	err = m.ParseGeography(body)
	if err != nil {
		return nil, err
	}

	// create a dedicated client for rpcache-aa host
	apiBaseUrl, err := m.Data.ApiURL("", nil)
	if err != nil {
		return nil, err
	}
	api := NewClient(apiBaseUrl.String())
	api.authToken = c.mainClient.authToken
	api.noSessionCookie = true // api server do not send auth tokens

	// get all forecasts available on the map
	u, err = m.ForecastURL()
	if err != nil {
		return nil, err
	}
	body, err = api.Get(u.String(), CacheDisabled)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	err = m.ParseMultiforecast(body)
	if err != nil {
		return nil, err
	}

	// get pictos
	if pictos != nil {
		err = pictos.Update(m.PictoNames(), c.mainClient)
		if err != nil {
			return nil, err
		}
	}
	return &m, nil
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

// GetAllMaps() fetches a map tree recursively
// * startPath : where to start the tree walk ("/" is the 'root' page)
// * pictos are stored in PictoStore
// * limit limits the number of maps downloaded
func (c *Crawler) GetAllMaps(startPath string, pictos PictoStore, limit int) (MapStore, error) {
	var (
		cnt  int
		maps = MapStore{}
	)
	type QueueItem struct {
		path   string
		parent *mfmap.MfMap
	}

	// root map has a nil parent
	queue := []QueueItem{{startPath, nil}}
	for {
		// stop when queue is empty or max count is reached
		i := len(queue) - 1
		if ((limit > 0) && (cnt >= limit)) || i < 0 {
			break
		}
		cnt++

		// pop queue and process next path
		next := queue[i]
		queue = queue[0:i]
		m, err := c.GetMap(next.path, next.parent, pictos)
		if err != nil {
			return nil, err
		}
		// enqueue children maps
		for _, sz := range m.Data.Subzones {
			queue = append(queue, QueueItem{sz.Path, m})
		}
		// store current map in the collection
		maps[m.Path()] = m
	}
	return maps, nil
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

// call internal ServeMux
func (c *MeteoContent) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	c.mux.ServeHTTP(resp, req)
}

// Register register itself to mux on "/" path
// should be registered last, after more specific paths like static assets
func (c *MeteoContent) Register(mux *http.ServeMux) {
	mux.Handle("/", c)
}
