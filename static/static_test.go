package static

import (
	"gometeo/appconf"
	"gometeo/testutils"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"text/template"
)

var expectedFiles = []struct {
	fs   fs.FS
	want string
}{
	{embedJS, "js/main.js"},
	{embedCSS, "css/meteo.css"},
	{embedFonts, "fonts/fa.woff2"},
	{embedFavicon, "favicon/favicon.svg"},
	{embedRobotsTxt, "robots.txt"},
}

func TestEmbeddedFiles(t *testing.T) {
	for _, test := range expectedFiles {
		err := fstest.TestFS(test.fs, test.want)
		if err != nil {
			t.Error(err)
		}
	}
}

var testStaticPaths = map[string][]string{
	"app": {
		"/js/{{.Id}}/main.js",
		"/js/{{.Id}}/components.js",
		"/css/{{.Id}}/meteo.css",
	},
	"vue": {
		"/js/{{.Id}}/vue.esm-browser.dev.js",
		"/js/{{.Id}}/vue.esm-browser.prod.js",
	},
	"highcharts": {
		"/js/{{.Id}}/highcharts.js",
		"/js/{{.Id}}/highcharts-more.js",
	},
	"leaflet": {
		"/js/{{.Id}}/leaflet.js",
		"/css/{{.Id}}/leaflet.css",
		"/css/{{.Id}}/images/layers.png",
		"/css/{{.Id}}/images/marker-icon.png", // etc...
	},
	"fonts": {
		"/fonts/{{.Id}}/fa.woff2",
	},
	"favicon": {
		"/favicon.ico",
		"/favicon.svg",
		"/apple-touch-icon.png",
		"/favicon-96x96.png",
		"/web-app-manifest-192x192.png",
		"/web-app-manifest-512x512.png",
		"/site.webmanifest",
	},
	"robots_txt": {
		"/robots.txt",
	},
}

func TestStaticHandler(t *testing.T) {

	mux := http.NewServeMux()
	Register(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	cl := srv.Client()

	for name, paths := range testStaticPaths {
		t.Run(name, func(t *testing.T) {
			for _, p := range paths {
				p = fillCacheId(t, p)
				testutils.CheckStatusCode(t, cl, srv.URL+p, http.StatusOK)
			}
		})
	}
}

// fillCacheId replaces {{.Id}} with appconf.CacheId() in s
func fillCacheId(t *testing.T, s string) string {
	data := struct {
		Id string
	}{
		Id: appconf.CacheId(),
	}
	var buf = &strings.Builder{}
	var tmpl = template.Must(template.New("").Parse(s))
	err := tmpl.Execute(buf, data)
	if err != nil {
		t.Error(err)
	}
	return buf.String()
}
