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
	var (
		err     error
		content *crawl.MeteoContent
	)
	// for dev/debug/test
	if cacheServer {
		content = crawl.LoadContent(cacheFile)
	}
	// fetch data if cache is disabled or failed
	if content == nil {
		crawler := crawl.NewCrawler()
		content, err = crawler.FetchAll("/", limit)
		if err != nil {
			return err
		}
		// for dev/debug/test
		if cacheServer {
			crawl.StoreContent(cacheFile, content)
		}
	}
	done := serveContent(addr, content)
	// wait for server termination
	return <-done
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

// startCrawler returns a "self-updating" MeteoContent
func startCrawler(startPath string, limit int) (*crawl.MeteoContent, <-chan (struct{})) {

	// concurrent use of http.Client is safe according to official documentation,
	// so we can share the same client for maps and pictos.
	cr := crawl.NewCrawler()

	// direct pipe the crawler output channel to MeteoContent.Receive()
	mapsChan := cr.Fetch(startPath, limit)

	// starts a goroutine to receive fetched maps
	// returns a chan to signal when mapsChan is closed
	content := crawl.NewContent()
	chCrawlerDone := content.Receive(mapsChan, cr)

	return content, chCrawlerDone
}

func Start(addr string, limit int) error {

	content, crawlerDone := startCrawler("/", limit)
	srvDone := serveContent(addr, content)

	// block until crawling stops
	select {
	case <-srvDone:
		{
			log.Printf("httpserver termination")
		}
	case <-crawlerDone:
		{
			log.Printf("crawler termination")
		}
	}

	return nil
}
