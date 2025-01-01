package server

import (
	"log"
	"net/http"

	"gometeo/crawl"
	"gometeo/static"
)

func NewMeteoMux(maps crawl.MapCollection, pics crawl.PictoStore) http.Handler {

	mux := http.ServeMux{}
	static.AddHandlers(&mux)
	pics.AddHandler(&mux)
	for _, m := range maps {
		m.AddHandlers(&mux)
	}
	hdl := wrapLogger(&mux)
	return hdl
}

func wrapLogger(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {

		// call the original http.Handler we're wrapping
		h.ServeHTTP(w, r)

		// gather information about request and log it
		//uri := r.URL.String()
		//method := r.Method
		// ... more information
		log.Println(r.URL.String())
	}
	// http.HandlerFunc wraps a function so that it
	// implements http.Handler interface
	return http.HandlerFunc(fn)
}

func StartSimple(maps crawl.MapCollection, pictos crawl.PictoStore) error {

	mux := NewMeteoMux(maps, pictos)
	err := http.ListenAndServe(":5151", mux)
	if err != http.ErrServerClosed {
		return err
	}
	return nil
}
