package crawl

import (
	"gometeo/mfmap"
	"log"
)

const (
	httpsMeteofranceCom = "https://meteofrance.com"
	sessionCookie       = "mfsession"
)

type Crawler struct {
	mainClient *MfClient
}

// NewCrawler allocates a Crawler with a pre-configured client
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

	// allocate a MfMap and initialize with received content
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

// GetAllMaps() fetches a map tree recursively
// * startPath : where to start the tree walk ("/" is the 'root' page)
// * pictos are stored in PictoStore
// * limit limits the number of maps downloaded
func (c *Crawler) FetchAll(startPath string, pictos PictoStore, limit int) (MapStore, error) {
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
