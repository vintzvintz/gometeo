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
	cacheServer = true
	cacheFile   = "./content_cache.gob"
)

// StartSimple fetches data once and serve it forever
func StartSimple(addr string) error {

	crawler := crawl.NewCrawler()
	var err error

	var content *crawl.MeteoContent

	if cacheServer {
		content = crawl.LoadContent(cacheFile)
	}

	// fetch data if cache is disabled or failed
	if content == nil {
		pictos := crawl.PictoStore{}
		maps, err := crawler.GetAllMaps("/", pictos, 15)
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
		content = crawl.NewContent()
		content.Update(maps, pictos)

		if cacheServer {
			crawl.StoreContent(cacheFile, content)
		}
	}

	srv := http.Server{
		Addr:    ":5151",
		Handler: makeMeteoHandler(content),
	}
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
