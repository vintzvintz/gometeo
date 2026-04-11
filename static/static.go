package static

import (
	"embed"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"net/http"

	"gometeo/obs"
)

const (
	JsPrefix    = "/js"
	CssPrefix   = "/css"
	FontsPrefix = "/fonts"
)

//go:embed js
var embedJS embed.FS

//go:embed css
var embedCSS embed.FS

//go:embed fonts
var embedFonts embed.FS

//go:embed favicon
var embedFavicon embed.FS

//go:embed robots.txt
var embedRobotsTxt embed.FS

func registerStatic(mux *http.ServeMux, prefix string, cacheId string, fs embed.FS, reg *obs.Registry) {

	// pattern matches URL of static ressources under prefix with cacheId
	pattern := prefix + "/" + cacheId + "/{filename...}"

	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		filename := r.PathValue("filename")
		fspath := prefix + "/" + filename
		w.Header().Add("Cache-Control", "max-age=31536000, immutable")
		http.ServeFileFS(w, r, fs, fspath)
		reg.RecordStaticServed()
	})
}

func Register(mux *http.ServeMux, cacheId string, reg *obs.Registry) {
	registerStatic(mux, JsPrefix, cacheId, embedJS, reg)
	registerStatic(mux, CssPrefix, cacheId, embedCSS, reg)
	registerStatic(mux, FontsPrefix, cacheId, embedFonts, reg)
	registerRobotsTxt(mux, reg)
	registerFavicon(mux, reg)
}

func registerRobotsTxt(mux *http.ServeMux, reg *obs.Registry) {
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, embedRobotsTxt, "robots.txt")
		reg.RecordStaticServed()
	})
}

var faviconContentTypes = map[string]string{
	"favicon.ico":                  "image/x-icon",
	"favicon.svg":                  "image/svg+xml",
	"apple-touch-icon.png":         "image/png",
	"favicon-96x96.png":            "image/png",
	"web-app-manifest-192x192.png": "image/png",
	"web-app-manifest-512x512.png": "image/png",
	"site.webmanifest":             "application/manifest+json",
}

func registerFavicon(mux *http.ServeMux, reg *obs.Registry) {
	fs.WalkDir(embedFavicon, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Fatal(err)
		}
		if d.IsDir() {
			return nil // ignore non-file entries
		}
		// register each file on root path
		mux.HandleFunc("/"+d.Name(), faviconHandlerFunc(path, d, reg))
		return nil
	})
}

func faviconHandlerFunc(path string, d fs.DirEntry, reg *obs.Registry) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		f, err := embed.FS.Open(embedFavicon, path)
		if err != nil {
			slog.Error("error opening embedded file", "path", path)
			return
		}
		mime, ok := faviconContentTypes[d.Name()]
		if !ok {
			slog.Warn("Content-Type unknown", "file", d.Name())
		} else {
			w.Header().Add("Content-Type", mime)
		}
		w.Header().Add("Cache-Control", "max-age=475440") // ́1 week
		w.WriteHeader(http.StatusOK)
		if _, err := io.Copy(w, f); err != nil {
			slog.Error("send error", "err", err)
		}
		reg.RecordStaticServed()
	}
}
