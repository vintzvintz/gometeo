package crawl

import (
	"bytes"
	"gometeo/mfmap"
	"io"
	"log"
	"net/http"
)

const (
	httpsMeteofranceCom = "https://meteofrance.com"
	sessionCookie       = "mfsession"
)

type Crawler struct {
	mainClient *MfClient
	apiClient  *MfClient
}

type MapCollection []*mfmap.MfMap

type PictoStore map[string][]byte

// NewCrawler allocates as *MfCrawler
func NewCrawler() *Crawler {
	return &Crawler{
		mainClient: NewClient(httpsMeteofranceCom),
		// apiClient: nil,  // apiClient needs API base url from main client
	}
}

// GetMap gets https://mf.com/zone html page and related data like
// svg map, pictos, forecasts and list of subzones
// related data is stored into MfMap fields
func (c *Crawler) GetMap(path string, parent *mfmap.MfMap, pictos PictoStore) (*mfmap.MfMap, error) {
	/*if parent==nil {
		log.Printf("GetMap() '%s' no parent", path)
	} else {
		log.Printf("GetMap() '%s' with parent '%s'", path, parent.Name())
	}*/
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
	c.apiClient = NewClient(apiBaseUrl.String())
	c.apiClient.authToken = c.mainClient.authToken
	c.apiClient.noSessionCookie = true // api server do not send auth tokens

	// get all forecasts available on the map
	u, err = m.ForecastURL()
	if err != nil {
		return nil, err
	}
	body, err = c.apiClient.Get(u.String(), CacheDisabled)
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
func (c *Crawler) GetAllMaps(startPath string, pictos PictoStore, limit int) (MapCollection, error) {
	var (
		cnt  int
		maps = make(MapCollection, 0, 200)
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
		if ((limit > 0) && (cnt > limit)) || i < 0 {
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
		maps = append(maps, m)
	}
	return maps, nil
}

func (ps PictoStore) AddHandler(mux *http.ServeMux) {
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
