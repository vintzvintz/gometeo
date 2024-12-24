package main

import (
	"encoding/gob"
	"log"
	"os"

	"gometeo/crawl"
	"gometeo/mfmap"
	"gometeo/server"
)

var (
	cacheMap  bool   = true
	cacheFile string = "./cachedMap.gob"
)

func loadMap() *mfmap.MfMap {

	if !cacheMap {
		return nil
	}

	f, err := os.Open(cacheFile)
	if err != nil {
		log.Println(err)
		return nil
	}

	dec := gob.NewDecoder(f)
	m := mfmap.MfMap{}
	err = dec.Decode(&m)
	if err != nil {
		log.Println(err)
		return nil
	}
	log.Printf("cacheMap enabled : map loaded from %s", cacheFile)
	return &m
}

func storeMap(m *mfmap.MfMap) {

	if !cacheMap {
		return
	}

	f, err := os.Create(cacheFile)
	if err != nil {
		panic(err)
	}
	enc := gob.NewEncoder(f)
	err = enc.Encode(m)
	if err != nil {
		panic(err)
	}
}

func main() {

	crawler := crawl.NewCrawler()
	var err error

	// for tests/debug
	m := loadMap()

	if m == nil {
		m, err = crawler.GetMap("/", nil)
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
		// for tests/debug
		storeMap(m)
	}

	err = server.StartSimple(server.MapCollection{m})
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
