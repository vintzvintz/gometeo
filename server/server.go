package server

import (
	"log"
	"net/http"
	"os"

	"gometeo/crawl"
	"gometeo/static"
)

// for dev/tests/debug
const (
	cacheServer = false
	cacheFile   = "./content_cache.gob"
)

// StartSimple fetches data once and serve it forever
func StartSimple(addr string) error {

	var (
		err     error
		content *crawl.MeteoContent
	)
	// dev/debug/test
	if cacheServer {
		content = crawl.LoadContent(cacheFile)
	}

	// fetch data if cache is disabled or failed
	if content == nil {
		crawler := crawl.NewCrawler()
		content, err = crawler.FetchAll("/", 15)
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}

		// dev/debug/test
		if cacheServer {
			crawl.StoreContent(cacheFile, content)
		}
	}

	srv := http.Server{
		Addr:    addr,
		Handler: makeMeteoHandler(content),
	}
	log.Printf("Start simple server on '%s'", addr)
	err = srv.ListenAndServe()
	if err != http.ErrServerClosed {
		return err
	}
	log.Printf("Server closed")
	return nil
}

func makeMeteoHandler(content *crawl.MeteoContent) http.Handler {
	mux := http.NewServeMux()
	static.Register(mux)
	content.Register(mux)
	hdl := withLogging(mux)
	return hdl
}
/*

func Start(addr string) error {

	content := crawl.NewContent()

	// signal when crawling terminates
	isCrawling := make( chan(struct{}) )

	// fetch maps and update content store forever
	go func() {
		crawler := crawl.NewCrawler()
		for item := range crawler.Fetch( "/", 15 ) {
			content.UpdateItem(item)
		}
		close(isCrawling)
	}()

	srv := http.Server{
		Addr:    addr,
		Handler: makeMeteoHandler(content),
	}
	err := srv.ListenAndServe()
	if err != http.ErrServerClosed {
		return err
	}

	// block until crawling stops
	//_ = <- isCrawling

	return nil
}*/


