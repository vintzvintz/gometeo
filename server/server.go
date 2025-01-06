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


// TODO: refactor in testutils
func loadContent() crawl.MeteoContent {
	content := crawl.NewContent()  // empty but non-nil 

	if !cacheServer {
		return content
	}
	f, err := os.Open(cacheFile)
	if err != nil {
		log.Println(err)
		return content
	}
	dec := gob.NewDecoder(f)
	err = dec.Decode(&content)
	if err != nil {
		log.Println(err)
		return content
	}
	log.Printf("content loaded from %s", cacheFile)
	return content
}

// TODO: refactor in testutils
func storeContent(content crawl.MeteoContent) {
	if !cacheServer {
		return
	}
	f, err := os.Create(cacheFile)
	if err != nil {
		panic(err)
	}
	enc := gob.NewEncoder(f)
	err = enc.Encode(content)
	if err != nil {
		panic(err)
	}
	log.Printf("content stored to %s", cacheFile)
}

func NewMeteoHandler(content crawl.MeteoContent) http.Handler {

	mux := http.ServeMux{}
	static.AddHandlers(&mux)
	content.Pictos.AddHandler(&mux)
	for _, m := range content.Maps {
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
	content := loadContent()

	if content.IsEmpty() {
		content, err = crawler.UpdateAll(content, "/", 15)
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}

		// for tests/debug
		storeContent(content)
	}

	mux := NewMeteoHandler(content)

	err = http.ListenAndServe(addr, mux)
	if err != http.ErrServerClosed {
		return err
	}
	return nil
}
/*
func (ms *MeteoServer) NewServer(addr string) {

}


func Start(addr string) error {





	return nil
}*/