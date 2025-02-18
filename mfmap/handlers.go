package mfmap

import (
	"bytes"
	"encoding/json"
	"gometeo/appconf"
	"io"
	"log"
	"net/http"

	gj "gometeo/geojson"
)

// Register adds handlers to mux for "/$path", "/$path/data", "/$path/svg"
// also a redirection from "/"" to "/france"
func (m *MfMap) Register(mux *http.ServeMux) {
	p := "/" + m.Path()
	// log.Printf("Register handlers for '%s'", p)
	mux.HandleFunc(p, m.makeMainHandler())
	mux.HandleFunc(p+"/data", m.makeDataHandler())
	mux.HandleFunc(p+"/"+appconf.CacheId()+"/svg", m.makeSvgMapHandler())
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
		resp.Header().Add("Content-Type", "text/html; charset=utf-8")
		resp.Header().Add("Cache-Control", "no-cache")
		resp.WriteHeader(http.StatusOK)
		_, err = io.Copy(resp, &buf)
		if err != nil {
			log.Printf("ignored send error: %s", err)
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
		resp.Header().Add("Cache-Control", "no-cache")
		resp.WriteHeader(http.StatusOK)
		_, err = io.Copy(resp, &buf)
		if err != nil {
			log.Printf("ignored send error: %s", err)
		}
		// update on data handler (JSON request) instead of main handler
		// to allow main page caching and avoid simplest bots
		m.MarkHit()
	}
}

func (m *MfMap) makeSvgMapHandler() func(http.ResponseWriter, *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		if len(m.SvgMap) == 0 {
			resp.WriteHeader(http.StatusNotFound)
			log.Printf("SVG map unavailable (req.URL='%s'", req.URL)
			return
		}
		resp.Header().Add("Cache-Control", "max-age=31536000, immutable")
		resp.Header().Add("Content-Type", "image/svg+xml")
		resp.WriteHeader(http.StatusOK)
		_, err := io.Copy(resp, bytes.NewReader(m.SvgMap))
		if err != nil {
			log.Printf("ignored send error: %s", err)
		}
	}
}

func makeRedirectHandler(url string) func(http.ResponseWriter, *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		log.Printf("redirect from %s to %s", req.URL, url)
		http.Redirect(resp, req, url, http.StatusMovedPermanently)
	}
}

type jsonMap struct {
	Name       string         `json:"name"`
	Path       string         `json:"path"`
	Breadcrumb Breadcrumbs    `json:"breadcrumb"`
	Idtech     string         `json:"idtech"`
	Taxonomy   string         `json:"taxonomy"`
	Bbox       gj.Bbox        `json:"bbox"`
	SubZones   gj.GeoFeatures `json:"subzones"`
	Prevs      gj.PrevList    `json:"prevs"`
	Chroniques gj.Graphdata   `json:"chroniques"`
}

// writes all forecast data available in m as a big json object
func (m *MfMap) WriteJson(wr io.Writer) error {
	obj, err := m.BuildJson()
	if err != nil {
		return err
	}
	b, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	_, err = io.Copy(wr, bytes.NewReader(b))
	if err != nil {
		return err
	}
	return nil
}

func (m *MfMap) BuildJson() (*jsonMap, error) {

	bbox := m.Geography.Bbox.Crop(
		cropPc.Left, cropPc.Right, cropPc.Top, cropPc.Bottom)

	j := jsonMap{
		Name:       m.Name(),
		Path:       m.Path(),
		Breadcrumb: m.Breadcrumb, // not from upstream
		Idtech:     m.Data.Info.IdTechnique,
		Taxonomy:   m.Data.Info.Taxonomy,
		SubZones:   m.Geography.Features,
		Bbox:       bbox,
		Prevs:      m.Prevs,
		Chroniques: m.Graphdata,
	}
	// highchart disabled for PAYS. Only on DEPTs & REGIONs
	if m.Data.Info.Taxonomy == "PAYS" {
		j.Chroniques = nil
	}
	return &j, nil
}
