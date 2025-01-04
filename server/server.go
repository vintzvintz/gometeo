package server

import (
	"encoding/gob"
	"log"
	"net/http"
	"os"

	"gometeo/crawl"
	"gometeo/static"
)

const (
	cacheServer = true
	cacheFile   = "./cachedServer.gob"
)

type MeteoServer struct {
	Maps   crawl.MapCollection
	Pictos crawl.PictoStore
}

// TODO: refactor in testutils
func loadServer() *MeteoServer {
	if !cacheServer {
		return nil
	}
	f, err := os.Open(cacheFile)
	if err != nil {
		log.Println(err)
		return nil
	}
	dec := gob.NewDecoder(f)
	srv := MeteoServer{}
	err = dec.Decode(&srv)
	if err != nil {
		log.Println(err)
		return nil
	}
	log.Printf("cacheMap enabled : map loaded from %s", cacheFile)
	return &srv
}

// TODO: refactor in testutils
func storeServer(srv *MeteoServer) {
	if !cacheServer {
		return
	}
	f, err := os.Create(cacheFile)
	if err != nil {
		panic(err)
	}
	enc := gob.NewEncoder(f)
	err = enc.Encode(*srv)
	if err != nil {
		panic(err)
	}
}

func NewMeteoHandler(maps crawl.MapCollection, pictos crawl.PictoStore) http.Handler {

	mux := http.ServeMux{}
	static.AddHandlers(&mux)
	pictos.AddHandler(&mux)
	for _, m := range maps {
		m.AddHandlers(&mux)
	}
	hdl := withLogging(&mux)
	return hdl
}

// StartSimple fetches data once and serve it forever
func StartSimple(addr string) error {

	crawler := crawl.NewCrawler()
	var err error

	// for tests/debug
	srv := loadServer()

	if srv == nil {
		p := crawl.PictoStore{}
		m, err := crawler.GetAllMaps("/", p, 15)
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
		srv = &MeteoServer{Maps: m, Pictos: p}

		// for tests/debug
		storeServer(srv)
	}

	mux := NewMeteoHandler(srv.Maps, srv.Pictos)
	err = http.ListenAndServe(addr, mux)
	if err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (ms *MeteoServer) NewServer(addr string) {

}
