package server

import (
	"bytes"
	"io"
	"log"
	"net/http"

	"gometeo/mfmap"
)

type MapCollection []*mfmap.MfMap

type MeteoServer *http.ServeMux

func NewMeteoMux(maps MapCollection) (*http.ServeMux, error) {
	mux := http.ServeMux{}
	for _, m := range maps {
		name, err := m.Name()
		if err != nil {
			return nil, err
		}
		log.Printf("Registering map '%s'", name)
		mux.HandleFunc("/"+name, makeMainHandler(m))

		// redirect root path '/' to '/france'
		if name == "france" {
			mux.HandleFunc("/", makeRedirectHandler("/france"))
		}
	}
	return &mux, nil
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

func makeRedirectHandler(url string) func(http.ResponseWriter, *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		log.Printf("redirect from %s to %s", req.URL, url)
		http.Redirect(resp, req, url, http.StatusMovedPermanently)
	}
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