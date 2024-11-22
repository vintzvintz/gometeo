package crawl

import (
	//"fmt"
	"gometeo/mfmap"
	"log"
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

// getMap gets a map from remote service or from local cache if available
func (c *MfCrawler) getMap(path string) (*mfmap.MfMap, error) {
	//log.Printf("Crawling %s from parent '%s'\n", path, parent.Nom)
	body, err := c.client.Get(path, CacheDefault)
	if err != nil {
		return nil, err
	}
	//m := &mfmap.MfMap{}
	m, err := mfmap.NewFrom(body)
	return m, err
}

func (c *MfCrawler) GetMap(path string, parent *mfmap.MfMap) (*mfmap.MfMap, error) {
	log.Printf("Crawling %s from parent '%s'\n", path, parent.Nom())
	m, err := c.getMap(path)
	if err != nil {
		return nil, err
	}
	m.SetParent(parent)
	// TODO m.nom = xxxx
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
