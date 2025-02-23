package server

import (
	"log"
	"net/http"
	"regexp"

	"gometeo/appconf"
	"gometeo/content"
	"gometeo/crawl"
	"gometeo/static"
)

// for dev/tests/debug
// TODO : refactor into a StartOneShot() parameter
const (
	cacheServer = true
	cacheFile   = "./content_cache.gob"
)

const startPath = "/"

func Start() error {
	addr := appconf.Addr()
	limit := appconf.Limit()

	entryPoint := startNormal
	if appconf.OneShot() {
		entryPoint = startOneShot
	}

	log.Printf(`Starting gometeo : Addr='%s' Limit=%d OneShot=%v `,
		addr, limit, appconf.OneShot())

	return entryPoint(addr, limit)
}

// StartSimple fetches data once (no updates)
// and serve it forever when done
func startOneShot(addr string, limit int) error {
	var c *content.Meteo

	// for dev/debug/test
	if cacheServer {
		c = content.LoadBlob(cacheFile)
	}
	// fetch data if cache is disabled or failed
	if c == nil {
		var crawlerDone <-chan struct{}
		c, crawlerDone = crawl.Start(startPath, limit, crawl.ModeOnce)
		<-crawlerDone // wait for all maps downloads to complete

		// for dev/debug/test
		if cacheServer {
			c.SaveBlob(cacheFile)
		}
	}
	_, serverDone := serveContent(addr, c)
	// wait for server termination
	return <-serverDone
}

// TODO add more crawl/serve/config options, maybe in a struct
func startNormal(addr string, limit int) error {

	c, crawlerDone := crawl.Start(startPath, limit, crawl.ModeForever)
	defer c.Close()

	srv, serverDone := serveContent(addr, c)
	defer srv.Close()

	// block until either server or crawler terminates
	select {
	case <-serverDone:
		log.Printf("server exited")
	case <-crawlerDone:
		log.Printf("crawler exited")
	}
	return nil
}

func makeMeteoHandler(mc *content.Meteo) http.Handler {
	mux := http.NewServeMux()
	static.Register(mux)
	mux.Handle("/", mc)
	hdl := withOldUrlRedirect(mux)
	hdl = withLogging(hdl)
	return hdl
}

func serveContent(addr string, mc *content.Meteo) (*http.Server, <-chan error) {
	srv := http.Server{
		Addr:    addr,
		Handler: makeMeteoHandler(mc),
	}
	ch := make(chan error)
	go func() {
		log.Printf("Start server on '%s'", addr)
		err := srv.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Printf("server error: %s", err)
		} else {
			log.Printf("Server closed")
		}
		ch <- err
		close(ch)
	}()
	return &srv, ch
}

func withOldUrlRedirect(h http.Handler) http.Handler {

	//
	pattern := regexp.MustCompile(`.*(/[^\/]+)\.html$`)

	redirectOld := func(resp http.ResponseWriter, req *http.Request) {

		// redirect legacy .html path
		match := pattern.FindStringSubmatch(req.URL.Path)
		if (match != nil) && (len(match) == 2) {
			newpath := match[1]
			log.Printf("LEGACY ADDRESS %s redirected to %s", req.URL, newpath)
			http.Redirect(resp, req, newpath, http.StatusMovedPermanently)
			return
		}

		// forward request to next handler
		h.ServeHTTP(resp, req)
	}

	return http.HandlerFunc(redirectOld)
}
