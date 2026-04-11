package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"gometeo/appconf"
	"gometeo/content"
	"gometeo/crawl"
	"gometeo/mfmap"
	"gometeo/mfmap/schedule"
)

const testDataDir = "../test_data/"

// testServerConf returns ServerConf values fast enough for unit tests.
func testServerConf() ServerConf {
	return ServerConf{
		FetchInterval:   50 * time.Millisecond,
		FetchTimeout:    2 * time.Second,
		ShutdownTimeout: 1 * time.Second,
	}
}

// newFastUpstream starts a fake upstream httptest.Server that serves the
// minimal fixtures needed for crawl.Fetch to succeed on startPath "/" with
// limit=1.  Returns the server URL and a transport that redirects all
// outgoing requests (regardless of original scheme/host) to the fake server.
// Both are cleaned up via t.Cleanup.
func newFastUpstream(t *testing.T) (serverURL string, transport http.RoundTripper) {
	t.Helper()

	racineHTML, err := os.ReadFile(testDataDir + "racine.html")
	if err != nil {
		t.Fatalf("read racine.html: %v", err)
	}
	geography, err := os.ReadFile(testDataDir + "geography.json")
	if err != nil {
		t.Fatalf("read geography.json: %v", err)
	}
	multiforecast, err := os.ReadFile(testDataDir + "multiforecast.json")
	if err != nil {
		t.Fatalf("read multiforecast.json: %v", err)
	}
	svgMap, err := os.ReadFile(testDataDir + "pays007.svg")
	if err != nil {
		t.Fatalf("read pays007.svg: %v", err)
	}
	minimalPicto := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><circle r="5"/></svg>`)
	cookie := &http.Cookie{Name: "mfsession", Value: "dummytoken"}

	mux := http.NewServeMux()

	// HTML page — returned for any non-asset path (including "/")
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, cookie)
		w.Header().Set("Content-Type", "text/html")
		w.Write(racineHTML)
	})

	// SVG map
	mux.HandleFunc("/modules/custom/mf_map_layers_v2/maps/desktop/METROPOLE/pays007.svg", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, cookie)
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Write(svgMap)
	})

	// Geography GeoJSON
	mux.HandleFunc("/modules/custom/mf_map_layers_v2/maps/desktop/METROPOLE/geo_json/pays007-aggrege.json", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, cookie)
		w.Header().Set("Content-Type", "application/json")
		w.Write(geography)
	})

	// Multiforecast API — the redirectingTransport maps the HTTPS API call
	// (https://rwg.meteofrance.com/internet2018client/2.0/multiforecast?...)
	// to this plain-HTTP handler.
	mux.HandleFunc("/internet2018client/2.0/multiforecast", func(w http.ResponseWriter, r *http.Request) {
		// The API client sets noSessionCookie=true so we must NOT set a cookie here.
		w.Header().Set("Content-Type", "application/json")
		w.Write(multiforecast)
	})

	// Pictos — any weather SVG icon
	mux.HandleFunc("/modules/custom/mf_tools_common_theme_public/svg/weather/", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, cookie)
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Write(minimalPicto)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return srv.URL, &redirectingTransport{target: srv.URL}
}

// redirectingTransport replaces the scheme and host of every outgoing request
// with those of target, preserving path and query.  This lets the crawler's
// HTTPS API calls reach a plain-HTTP httptest.Server.
type redirectingTransport struct {
	target string
}

func (rt *redirectingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	u, err := url.Parse(rt.target)
	if err != nil {
		return nil, fmt.Errorf("redirectingTransport: bad target: %w", err)
	}
	clone := req.Clone(req.Context())
	clone.URL.Scheme = u.Scheme
	clone.URL.Host = u.Host
	clone.Host = u.Host
	return http.DefaultTransport.RoundTrip(clone)
}

// makeCrawlConf returns a CrawlConf pointing at upstream with the given transport.
func makeCrawlConf(upstream string, transport http.RoundTripper) crawl.CrawlConf {
	return crawl.CrawlConf{
		Upstream:  upstream,
		Transport: transport,
		MapConf: mfmap.MapConf{
			CacheId:  appconf.CacheId(),
			VueJs:    appconf.VueJs(),
			Upstream: upstream,
			Rates: schedule.UpdateRates{
				HotDuration: 72 * time.Hour,
				HotMaxAge:   60 * time.Minute,
				ColdMaxAge:  240 * time.Minute,
			},
		},
	}
}

// pollHealthz polls GET /healthz on addr until 200 or timeout.
func pollHealthz(t *testing.T, addr string, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	cl := &http.Client{Timeout: 300 * time.Millisecond}
	for time.Now().Before(deadline) {
		resp, err := cl.Get("http://" + addr + "/healthz")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return true
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

// startNormalForTest starts the normal-mode server loop in a goroutine using
// the provided crawl conf and a pre-bound ":0" listener.
// Returns the bound address and a channel that receives startNormal's return value.
func startNormalForTest(
	t *testing.T,
	ctx context.Context,
	sconf ServerConf,
	cc crawl.CrawlConf,
	limit int,
) (addr string, done <-chan error) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr = ln.Addr().String()
	ch := make(chan error, 1)

	go func() {
		cr := crawl.NewCrawler(cc)
		c := content.New(contentConf(nil))
		defer c.Close()

		initCtx, cancelInit := context.WithTimeout(ctx, sconf.FetchTimeout)
		chMap, chPicto := cr.Fetch(initCtx, startPath, limit)
		initDone := c.Receive(chMap, chPicto)

		crawlerDone := make(chan struct{})
		go func() {
			defer close(crawlerDone)
			defer cancelInit()
			runUpdateLoop(ctx, sconf, cr, c, initDone)
		}()

		srv, serverDone := serveContentOn(ln, c)
		select {
		case <-ctx.Done():
			shutdownServer(srv, sconf.ShutdownTimeout)
			<-serverDone
			ch <- nil
		case err := <-serverDone:
			ch <- err
		case <-crawlerDone:
			shutdownServer(srv, sconf.ShutdownTimeout)
			<-serverDone
			ch <- nil
		}
	}()

	return addr, ch
}

// startOneShotForTest starts the oneshot-mode server loop in a goroutine.
func startOneShotForTest(
	t *testing.T,
	ctx context.Context,
	sconf ServerConf,
	cc crawl.CrawlConf,
	limit int,
) (addr string, done <-chan error) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr = ln.Addr().String()
	ch := make(chan error, 1)

	go func() {
		cr := crawl.NewCrawler(cc)
		c := content.New(contentConf(nil))

		fetchCtx, cancel := context.WithTimeout(ctx, sconf.FetchTimeout)
		chMap, chPicto := cr.Fetch(fetchCtx, startPath, limit)
		<-c.Receive(chMap, chPicto)
		cancel()

		srv, serverDone := serveContentOn(ln, c)
		select {
		case <-ctx.Done():
			shutdownServer(srv, sconf.ShutdownTimeout)
			<-serverDone
			ch <- nil
		case err := <-serverDone:
			ch <- err
		}
	}()

	return addr, ch
}

// recordingHandler is an slog.Handler that records all log messages.
// It does NOT forward to any underlying handler to avoid re-entrancy issues.
type recordingHandler struct {
	records []slog.Record
}

func (h *recordingHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }
func (h *recordingHandler) Handle(_ context.Context, r slog.Record) error {
	h.records = append(h.records, r)
	return nil
}
func (h *recordingHandler) WithAttrs(_ []slog.Attr) slog.Handler  { return h }
func (h *recordingHandler) WithGroup(_ string) slog.Handler        { return h }

func (h *recordingHandler) hasMessage(needle string) bool {
	for _, r := range h.records {
		if strings.Contains(r.Message, needle) {
			return true
		}
	}
	return false
}

// withRecordingLogger replaces the default slog logger for the duration of
// the test and returns the recording handler.
func withRecordingLogger(t *testing.T) *recordingHandler {
	t.Helper()
	orig := slog.Default()
	rh := &recordingHandler{}
	slog.SetDefault(slog.New(rh))
	t.Cleanup(func() { slog.SetDefault(orig) })
	return rh
}

// init ensures appconf is initialized when the test binary runs.
func init() {
	if strings.Contains(strings.Join(os.Args, " "), "-test.") {
		appconf.Init([]string{})
	}
}
