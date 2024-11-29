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

type MfCrawler struct {
	client *MfClient
}

// NewCrawler allocates as *MfCrawler
func NewCrawler() *MfCrawler {
	return &MfCrawler{
		client: NewClient(),
	}
}

// GetMap gets https://mf.com/zone html page and related data like
// svg map, pictos, forecasts and list of subzones
// related data is stored into MfMap fields
func (c *MfCrawler) GetMap(zone string, parent *mfmap.MfMap) (*mfmap.MfMap, error) {
	//log.Printf("Crawling %s from parent '%s'\n", path, parent.Nom())
	//m, err := c.getMap(path)

	body, err := c.client.Get(zone, CacheDefault)
	if err != nil {
		return nil, err
	}
	m := &mfmap.MfMap{
		//		Nom: nom,
		Parent: parent,
	}
	err = m.ParseHtml(body)
	if err != nil {
		return nil, err
	}
	return m, nil
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
