package server

import (
	"bytes"
	"io"
	"log"
	"net/http"

	"gometeo/mfmap"
	"gometeo/static"
)

type MapCollection []*mfmap.MfMap

type MeteoServer *http.ServeMux

func NewMeteoMux(maps MapCollection) (http.Handler, error) {
	mux := http.ServeMux{}

	static.AddHandlers(&mux)

	for _, m := range maps {
		name, err := m.Name()
		if err != nil {
			return nil, err
		}
		log.Printf("Registering map '%s'", name)
		mux.HandleFunc("/"+name, makeMainHandler(m))
		mux.HandleFunc("/"+name+"/data", makeDataHandler(m))

		// redirect root path '/' to '/france'
		if name == "france" {
			mux.HandleFunc("/{$}", makeRedirectHandler("/france/"))
		}
	}

	hdl := wrapLogger(&mux)
	return hdl, nil
}

// makeMainHandler wraps a map into an handler function
func makeMainHandler(m *mfmap.MfMap) func(http.ResponseWriter, *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		// do not stream directly to resp before knowing error status
		buf := bytes.Buffer{}
		err := m.BuildHtml(&buf)
		if err != nil {
			resp.WriteHeader(http.StatusInternalServerError)
			log.Printf("BuildHtml on req '%s' error: %s", req.URL, err)
			return
		}
		n, err := io.Copy(resp, &buf)
		if err != nil {
			log.Printf("send error: %s", err)
		}
		log.Printf("GET %s sent %d bytes", req.URL, n)
	}
}

func makeDataHandler(m *mfmap.MfMap) func(http.ResponseWriter, *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		err := m.BuildJson(resp)
		if err != nil {
			resp.WriteHeader(http.StatusInternalServerError)
			log.Printf("BuildHtml on req '%s' error: %s", req.URL, err)
			return
		}
	}
}

func makeRedirectHandler(url string) func(http.ResponseWriter, *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		log.Printf("redirect from %s to %s", req.URL, url)
		http.Redirect(resp, req, url, http.StatusMovedPermanently)
	}
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

func StartSimple(maps MapCollection) error {

	mux, err := NewMeteoMux(maps)
	if err != nil {
		return err
	}
	err = http.ListenAndServe(":5151", mux)
	if err != http.ErrServerClosed {
		return err
	}
	return nil
}
