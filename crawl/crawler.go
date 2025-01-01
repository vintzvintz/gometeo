package crawl

import (
	//"fmt"
	"gometeo/mfmap"
	//"log"
)

const (
	httpsMeteofranceCom = "https://meteofrance.com"
	sessionCookie       = "mfsession"
)

type Crawler struct {
	mainClient *MfClient
	apiClient  *MfClient
}


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
func (c *Crawler) GetMap(zone string, parent *mfmap.MfMap) (*mfmap.MfMap, error) {
	//log.Printf("Crawling %s from parent '%s'\n", path, parent.Nom())
	//m, err := c.getMap(path)

	body, err := c.mainClient.Get(zone, CacheDisabled)
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
	return &m, nil
}


func (c *Crawler) Pictos() PictoStore {

	store := make( PictoStore)

	return  store
}


func SampleRun(path string) error {
	crawler := NewCrawler()
	m, err := crawler.GetMap(path, nil)
	if err != nil {
		return err
	}
	_ = m

	/*
		html := m.html
		var trunc int = min(int(200), len(html))
		fmt.Printf(html[0:trunc])
	*/
	return nil
}
