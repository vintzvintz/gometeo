package server

import (
	"log"
	"net/http"

	"gometeo/crawl"
	"gometeo/static"
)

// for dev/tests/debug
const (
	cacheServer = true
	cacheFile   = "./content_cache.gob"
)

// StartSimple fetches data once and serve it forever
func StartSimple(addr string, limit int) error {
	var content *crawl.MeteoContent

	// for dev/debug/test
	if cacheServer {
		content = crawl.LoadBlob(cacheFile)
	}
	// fetch data if cache is disabled or failed

	if content == nil {
		var crawlerDone <-chan struct{}
		content, crawlerDone = crawl.Start("/", limit, crawl.ModeOnce)
		<-crawlerDone // wait for all maps downloads to complete

		// for dev/debug/test
		if cacheServer {
			content.SaveBlob(cacheFile)
		}
	}
	serverDone := serveContent(addr, content)
	// wait for server termination
	return <-serverDone
}

func makeMeteoHandler(content *crawl.MeteoContent) http.Handler {
	mux := http.NewServeMux()
	static.Register(mux)
	content.Register(mux)
	hdl := withLogging(mux)
	return hdl
}

func serveContent(addr string, content *crawl.MeteoContent) <-chan error {
	srv := http.Server{
		Addr:    addr,
		Handler: makeMeteoHandler(content),
	}
	ch := make(chan error)
	go func() {
		log.Printf("Start server on '%s'", addr)
		err := srv.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Printf("server error: %s", err)
		} else {
			log.Printf("Server closed")
		}
		ch <- err
		close(ch)
	}()
	return ch
}

/*
func Start(addr string, limit int) error {

	content, crawlerDone := crawl.CreateAndStart("/", limit)
	srvDone := serveContent(addr, content)

	// block until crawling stops
	select {
	case <-srvDone:
			log.Printf("httpserver termination")
	case <-crawlerDone:
			log.Printf("crawler termination")
	}
	return nil
}
*/
