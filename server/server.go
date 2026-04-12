package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"gometeo/appconf"
	"gometeo/content"
	"gometeo/crawl"
	"gometeo/mfmap"
	"gometeo/mfmap/schedule"
	"gometeo/obs"
	"gometeo/static"
)

const startPath = "/"

// ServerConf holds server-level tuning parameters.
// It is internal to the server package; tests inject custom values directly.
type ServerConf struct {
	FetchInterval   time.Duration
	FetchTimeout    time.Duration
	ShutdownTimeout time.Duration
}

func defaultServerConf() ServerConf {
	return ServerConf{
		FetchInterval:   10 * time.Second,
		FetchTimeout:    5 * time.Minute,
		ShutdownTimeout: 10 * time.Second,
	}
}

func contentConf(reg *obs.Registry) content.ContentConf {
	dayMin, dayMax := appconf.KeepDays()
	return content.ContentConf{
		DayMin:  dayMin,
		DayMax:  dayMax,
		CacheId: appconf.CacheId(),
		Obs:     reg,
	}
}

func crawlConf(reg *obs.Registry) crawl.CrawlConf {
	return crawl.CrawlConf{
		Upstream: appconf.Upstream(),
		MapConf:  mapConf(),
		Obs:      reg,
	}
}

func mapConf() mfmap.MapConf {
	r := appconf.UpdateRate()
	return mfmap.MapConf{
		CacheId:  appconf.CacheId(),
		VueJs:    appconf.VueJs(),
		Upstream: appconf.Upstream(),
		Rates: schedule.UpdateRates{
			HotDuration:    r.HotDuration,
			HotMaxAge:      r.HotMaxAge,
			ColdMaxAge:     r.ColdMaxAge,
			FailureBackoff: r.FailureBackoff,
		},
	}
}

func Start() error {
	// Remove timestamp from slog output to avoid duplication with journald timestamps.
	// Errors and warnings go to stderr; info and debug go to stdout.
	logOpts := &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}
	slog.SetDefault(slog.New(newLevelSplitHandler(os.Stdout, os.Stderr, logOpts)))

	rates := appconf.UpdateRate()
	slog.Info("starting gometeo", "commit", appconf.Commit(), "addr", appconf.Addr(), "limit", appconf.Limit(), "oneshot", appconf.OneShot(), "vuejs", appconf.VueJs())
	slog.Info("update rates", "hotDuration", rates.HotDuration, "hotMaxAge", rates.HotMaxAge, "coldMaxAge", rates.ColdMaxAge, "failureBackoff", rates.FailureBackoff)

	// Root context cancelled on SIGINT/SIGTERM for graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return startWithContext(ctx, defaultServerConf(), obs.NewRegistry())
}

func startWithContext(ctx context.Context, sconf ServerConf, reg *obs.Registry) error {
	addr := appconf.Addr()
	limit := appconf.Limit()

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}

	if appconf.OneShot() {
		return startOneShot(ctx, sconf, ln, limit, reg)
	}
	return startNormal(ctx, sconf, ln, limit, reg)
}

// startOneShot fetches data once (no updates) and serves it forever when done.
func startOneShot(ctx context.Context, sconf ServerConf, ln net.Listener, limit int, reg *obs.Registry) error {
	var c *content.Meteo

	cacheFile := appconf.CacheFile()
	if cacheFile != "" {
		c = content.LoadBlob(cacheFile, contentConf(reg), mapConf())
	}
	// fetch data if cache is disabled or failed
	if c == nil {
		fetchCtx, cancel := context.WithTimeout(ctx, sconf.FetchTimeout)
		cr := crawl.NewCrawler(crawlConf(reg))
		c = content.New(contentConf(reg))
		chMap, chPicto := cr.Fetch(fetchCtx, startPath, limit)
		<-c.Receive(chMap, chPicto) // wait for all maps downloads to complete
		cancel()

		if cacheFile != "" {
			if err := c.SaveBlob(cacheFile); err != nil {
				slog.Error("SaveBlob error", "err", err)
			}
		}
	}
	srv, serverDone := serveContentOn(ln, c)
	// shut down on signal or wait for server termination
	select {
	case <-ctx.Done():
		shutdownServer(srv, sconf.ShutdownTimeout)
		<-serverDone
		return nil
	case err := <-serverDone:
		return err
	}
}

func startNormal(ctx context.Context, sconf ServerConf, ln net.Listener, limit int, reg *obs.Registry) error {
	cr := crawl.NewCrawler(crawlConf(reg))
	c := content.New(contentConf(reg))
	defer c.Close()

	// initial fetch, bounded by FetchTimeout so startup can't hang forever
	initCtx, cancelInit := context.WithTimeout(ctx, sconf.FetchTimeout)
	chMap, chPicto := cr.Fetch(initCtx, startPath, limit)
	initDone := c.Receive(chMap, chPicto)

	// forever update loop in background
	crawlerDone := make(chan struct{})
	go func() {
		defer close(crawlerDone)
		defer cancelInit()
		runUpdateLoop(ctx, sconf, cr, c, initDone)
	}()

	srv, serverDone := serveContentOn(ln, c)

	// block until signal, server exit, or crawler exit
	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
		shutdownServer(srv, sconf.ShutdownTimeout)
		<-serverDone
	case err := <-serverDone:
		slog.Info("server exited", "err", err)
	case <-crawlerDone:
		slog.Info("crawler exited")
		shutdownServer(srv, sconf.ShutdownTimeout)
		<-serverDone
	}
	return nil
}

// runUpdateLoop waits for initDone then repeatedly fetches the next map
// needing an update. It exits when ctx is cancelled.
func runUpdateLoop(
	ctx context.Context,
	sconf ServerConf,
	cr *crawl.Crawler,
	c *content.Meteo,
	initDone <-chan struct{},
) {
	<-initDone
	slog.Info("enter forever update loop")
	ticker := time.NewTicker(sconf.FetchInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		path := c.Updatable()
		if path == "" {
			continue
		}
		fetchCtx, cancel := context.WithTimeout(ctx, sconf.FetchTimeout)
		chMap, chPicto := cr.Fetch(fetchCtx, path, 1)
		// Tee the map channel so we can tell whether the fetch produced a map.
		// On failure, mark the map so the scheduler applies the failure backoff
		// and we don't hammer upstream every tick.
		teedMap := make(chan *mfmap.MfMap)
		received := 0
		go func() {
			defer close(teedMap)
			for m := range chMap {
				received++
				teedMap <- m
			}
		}()
		<-c.Receive(teedMap, chPicto)
		if received == 0 {
			c.MarkFailure(path)
		}
		cancel()
	}
}

// shutdownServer attempts a graceful shutdown with a bounded deadline,
// falling back to Close() if the deadline is exceeded.
func shutdownServer(srv *http.Server, timeout time.Duration) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Warn("graceful shutdown failed, forcing close", "err", err)
		srv.Close()
	}
}

func makeMeteoHandler(mc *content.Meteo) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if mc.Ready() {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "ok")
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintln(w, "not ready")
		}
	})
	static.Register(mux, appconf.CacheId(), mc.Obs())
	mux.Handle("/", mc)
	hdl := withOldUrlRedirect(mux)
	hdl = withLogging(hdl)
	return hdl
}

// serveContentOn starts an HTTP server on the provided listener.
// Tests use this with a ":0" listener to get a random port.
func serveContentOn(ln net.Listener, mc *content.Meteo) (*http.Server, <-chan error) {
	srv := &http.Server{Handler: makeMeteoHandler(mc)}
	ch := make(chan error, 1)
	go func() {
		slog.Info("start server", "addr", ln.Addr())
		err := srv.Serve(ln)
		if err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
		} else {
			slog.Info("server closed")
		}
		ch <- err
		close(ch)
	}()
	return srv, ch
}

func withOldUrlRedirect(h http.Handler) http.Handler {

	//
	pattern := regexp.MustCompile(`.*(/[^\/]+)\.html$`)

	redirectOld := func(resp http.ResponseWriter, req *http.Request) {

		// redirect legacy .html path
		match := pattern.FindStringSubmatch(req.URL.Path)
		if (match != nil) && (len(match) == 2) {
			newpath := match[1]
			slog.Info("legacy address redirect", "from", req.URL, "to", newpath)
			http.Redirect(resp, req, newpath, http.StatusMovedPermanently)
			return
		}

		// forward request to next handler
		h.ServeHTTP(resp, req)
	}

	return http.HandlerFunc(redirectOld)
}
