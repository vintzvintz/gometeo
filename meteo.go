package main

import (
	"encoding/gob"
	"log"
	"os"

	"gometeo/crawl"
	"gometeo/server"
)

var (
	cacheMaps bool   = true
	cacheFile string = "./cachedMaps.gob"
)

type MeteoBlob struct {
	Maps   crawl.MapCollection
	Pictos crawl.PictoStore
}

// TODO: refactor in testutils
func loadMaps() (crawl.MapCollection, crawl.PictoStore) {
	if !cacheMaps {
		return nil, nil
	}
	f, err := os.Open(cacheFile)
	if err != nil {
		log.Println(err)
		return nil, nil
	}
	dec := gob.NewDecoder(f)
	blob := MeteoBlob{}
	err = dec.Decode(&blob)
	if err != nil {
		log.Println(err)
		return nil, nil
	}
	log.Printf("cacheMap enabled : map loaded from %s", cacheFile)
	return blob.Maps, blob.Pictos
}

// TODO: refactor in testutils
func storeMaps(maps crawl.MapCollection, pictos crawl.PictoStore) {
	if !cacheMaps {
		return
	}
	f, err := os.Create(cacheFile)
	if err != nil {
		panic(err)
	}
	enc := gob.NewEncoder(f)
	err = enc.Encode(MeteoBlob{maps, pictos})
	if err != nil {
		panic(err)
	}
}

func main() {

	crawler := crawl.NewCrawler()
	var err error

	// for tests/debug
	maps, pictos := loadMaps()

	if maps == nil {
		pictos = crawl.PictoStore{}
		maps, err = crawler.GetAllMaps("/", pictos, 0)
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
		// for tests/debug
		storeMaps(maps, pictos)
	}

	err = server.StartSimple(maps, pictos)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
