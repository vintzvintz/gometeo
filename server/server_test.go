package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"gometeo/crawl"
	"gometeo/mfmap"
)

// TODO : use local test data to avoid external requests
// should refactor buildTestMap() out of mfmap/build_test.go
func getMapTest(t *testing.T, path string) *mfmap.MfMap {
	m, err := crawl.NewCrawler().GetMap(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestMainHandler(t *testing.T) {
	m := getMapTest(t, "/")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()

	hdl := makeMainHandler(m)
	hdl(resp, req)

	// todo check response
	resp.Result()
}

func TestSvgHandler(t *testing.T) {
	m := getMapTest(t, "/")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	record := httptest.NewRecorder()

	hdl := makeSvgHandler(m)
	hdl(record, req)

	// todo check response
	body, err := io.ReadAll(record.Result().Body)
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`<svg`)
	if !re.Match(body) {
		t.Fatalf("missing <svg> tag")
	}
}

var testUrls = map[string][]string{
	"homepage": {
		"/",
		"/france",
	},
	"js": {
		"/js/main.js",
		"/js/vue.esm-browser.js",
		"/js/highcharts.js",
	},
	"css": {
		"/css/meteo.css",
	},
	"leaflet": {
		"/js/leaflet.js",
		"/css/leaflet.css",
		"/css/images/layers.png",
		"/css/images/marker-icon.png", // etc...
	},
	"fonts": {
		"/fonts/fa.woff2",
	},
}

func TestServer(t *testing.T) {
	maps := MapCollection{
		getMapTest(t, "/"),
	}

	srv := spawnServer(t, maps)
	defer srv.Close()
	cl := srv.Client()

	for name, urls := range testUrls {
		t.Run(name, func(t *testing.T) {
			for _, u := range urls {
				testPathOk(t, cl, srv.URL+u)
			}
		})
	}
}

func spawnServer(t *testing.T, maps MapCollection) *httptest.Server {
	mux, err := NewMeteoMux(maps)
	if err != nil {
		t.Fatalf("NewMeteoMux() error: %s", err)
	}
	return httptest.NewServer(mux)
}

func testPathOk(t *testing.T, cl *http.Client, path string) {
	resp, err := cl.Get(path)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status code %d expect %d", resp.StatusCode, http.StatusOK)
	}
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
}
