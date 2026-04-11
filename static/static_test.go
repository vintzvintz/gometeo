package static

import (
	"gometeo/testutils"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"text/template"
)

const testCacheId = "testcache"

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

// testStaticPaths maps a group name to a list of [path, needle] pairs.
// path is a URL (with {{.Id}} placeholder for the cache id).
// needle, if non-empty, must appear in the response body.
var testStaticPaths = map[string][][2]string{
	"app": {
		{"/js/{{.Id}}/main.js", "createApp"},
		{"/js/{{.Id}}/components.js", "mapComponents"},
		{"/css/{{.Id}}/meteo.css", "html"},
	},
	"vue": {
		{"/js/{{.Id}}/vue.esm-browser.dev.js", "vue v3.5.32"},
		{"/js/{{.Id}}/vue.esm-browser.prod.js", "vue v3.5.32"},
	},
	"highcharts": {
		{"/js/{{.Id}}/highcharts.js", "Highcharts JS v12.4.0"},
		{"/js/{{.Id}}/highcharts-more.js", "highcharts/highcharts-more"},
	},
	"leaflet": {
		{"/js/{{.Id}}/leaflet.js", "Leaflet 1.9.4"},
		{"/css/{{.Id}}/leaflet.css", "leaflet-pane"},
		{"/css/{{.Id}}/images/layers.png", ""},
		{"/css/{{.Id}}/images/marker-icon.png", ""},
	},
	"fonts": {
		{"/fonts/{{.Id}}/fa.woff2", ""},
	},
	"favicon": {
		{"/favicon.ico", ""},
		{"/favicon.svg", ""},
		{"/apple-touch-icon.png", ""},
		{"/favicon-96x96.png", ""},
		{"/web-app-manifest-192x192.png", ""},
		{"/web-app-manifest-512x512.png", ""},
		{"/site.webmanifest", ""},
	},
	"robots_txt": {
		{"/robots.txt", "User-Agent"},
	},
}

func TestStaticHandler(t *testing.T) {

	mux := http.NewServeMux()
	Register(mux, testCacheId, nil)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	cl := srv.Client()

	for name, paths := range testStaticPaths {
		t.Run(name, func(t *testing.T) {
			for _, pair := range paths {
				path := fillCacheId(t, pair[0])
				needle := pair[1]
				url := srv.URL + path
				testutils.CheckStatusCode(t, cl, url, http.StatusOK)
				if needle == "" {
					continue
				}
				resp, err := cl.Get(url)
				if err != nil {
					t.Errorf("%s: %v", path, err)
					continue
				}
				buf := make([]byte, 512)
				n, _ := resp.Body.Read(buf)
				resp.Body.Close()
				if !strings.Contains(string(buf[:n]), needle) {
					t.Errorf("%s: %q not found in first %d bytes", path, needle, n)
				}
			}
		})
	}
}

// fillCacheId replaces {{.Id}} with appconf.CacheId() in s
func fillCacheId(t *testing.T, s string) string {
	data := struct {
		Id string
	}{
		Id: testCacheId,
	}
	var buf = &strings.Builder{}
	var tmpl = template.Must(template.New("").Parse(s))
	err := tmpl.Execute(buf, data)
	if err != nil {
		t.Error(err)
	}
	return buf.String()
}
