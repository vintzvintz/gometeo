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
	apiClient *MfClient
}

// NewCrawler allocates as *MfCrawler
func NewCrawler() *Crawler {
	return &Crawler{
		mainClient: NewClient( httpsMeteofranceCom ),
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

	// get svg map with geography data
	u, err := m.SvgURL()
	if err != nil {
		return nil, err
	}
	body, err = c.mainClient.Get(u.String(), CacheDisabled)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	err = m.ParseSvgMap(body)
	if err != nil {
		return nil, err
	}
/*
	u, err := m.SvgURL()
	if err != nil {
		return nil, err
	}
	body, err = c.client.Get(u.String(), CacheDisabled)
	if err != nil {
		return nil, err
	}
	defer body.Close()


*/


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
	m.ParseMultiforecast(body)
	return &m, nil
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
