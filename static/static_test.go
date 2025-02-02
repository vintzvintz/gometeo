package static

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"gometeo/testutils"
)

var expectedFiles = map[string]struct {
	fs    fs.FS
	files []string
}{
	"js": {
		embedJS,
		[]string{"highcharts.js", "highcharts-more.js", "meteo.js"},
	},
	"css": {
		embedCSS,
		[]string{"meteo.css"},
	},
	"fonts": {
		embedFonts,
		[]string{"fa.woff2"},
	},
}

func TestEmbeddedFiles(t *testing.T) {
	for dir, files := range expectedFiles {
		t.Run(dir, func(t *testing.T) {
			for _, f := range files.files {
				want := dir + "/" + f
				err := fstest.TestFS(files.fs, want)
				if err != nil {
					t.Error(err)
				}
			}
		})
	}
}

var testStaticPaths = map[string][]string{
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
	"/favicon": {
	"/favicon.ico",
	"/favicon.svg",
	"/apple-touch-icon.png",
	"/favicon-96x96.png",
	"/web-app-manifest-192x192.png",
	"/web-app-manifest-512x512.png",
	"/site.webmanifest",
	},
}

func TestStaticHandler(t *testing.T) {

	mux := http.NewServeMux()
	Register(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	cl := srv.Client()

	for name, urls := range testStaticPaths {
		t.Run(name, func(t *testing.T) {
			for _, u := range urls {
				testutils.CheckStatusCode(t, cl, srv.URL+u, http.StatusOK)
			}
		})
	}
}
