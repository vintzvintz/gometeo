package mfmap

import (
	"bytes"
	"io"
	"log"
	"net/http"
)


// Register adds handlers to mux for "/$path", "/$path/data", "/$path/svg"
// also a redirection from "/"" to "/france"
func (m *MfMap) Register(mux *http.ServeMux) {
	p := "/"+m.Path()
//	log.Printf("Register handlers for '%s'", p)
	mux.HandleFunc(p, m.makeMainHandler())
	mux.HandleFunc(p+"/data", m.makeDataHandler())
	mux.HandleFunc(p+"/svg", m.makeSvgMapHandler())
	if p == "/france" {
		mux.HandleFunc("/{$}", makeRedirectHandler("/france"))
	}
}

// makeMainHandler wraps a map into an handler function
func (m *MfMap) makeMainHandler() func(http.ResponseWriter, *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		// do not stream directly to resp before knowing error status
		buf := bytes.Buffer{}
		err := m.WriteHtml(&buf)
		if err != nil {
			resp.WriteHeader(http.StatusInternalServerError)
			log.Printf("BuildHtml on req '%s' error: %s", req.URL, err)
			return
		}
		_, err = io.Copy(resp, &buf)
		if err != nil {
			log.Printf("send error: %s", err)
		}
	}
}

func (m *MfMap) makeDataHandler() func(http.ResponseWriter, *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		buf := bytes.Buffer{}
		err := m.WriteJson(&buf)
		if err != nil {
			resp.WriteHeader(http.StatusInternalServerError)
			log.Printf("BuildHtml on req '%s' error: %s", req.URL, err)
			return
		}
		resp.Header().Add("Content-Type", "application/json")
		_, err = io.Copy(resp, &buf)
		if err != nil {
			log.Printf("send error: %s", err)
		}
		// update on data handler (JSON request) instead of main handler 
		// to allow main page caching and avoid simplest bots
		m.LogHit()  
	}
}

func (m *MfMap) makeSvgMapHandler() func(http.ResponseWriter, *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {

		if len(m.SvgMap) == 0 {
			resp.WriteHeader(http.StatusNotFound)
			log.Printf("SVG map unavailable (req.URL='%s'", req.URL)
			return
		}
		resp.Header().Add("Content-Type", "image/svg+xml")
		_, err := io.Copy(resp, bytes.NewReader(m.SvgMap))
		if err != nil {
			log.Printf("send error: %s", err)
		}
	}
}

func makeRedirectHandler(url string) func(http.ResponseWriter, *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		log.Printf("redirect from %s to %s", req.URL, url)
		http.Redirect(resp, req, url, http.StatusMovedPermanently)
	}
}
