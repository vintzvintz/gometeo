package crawl

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"gometeo/mfmap"
	"gometeo/mfmap/urls"
	"gometeo/obs"
)

const (
	sessionCookie = "mfsession"
)

type CrawlConf struct {
	Upstream  string
	MapConf   mfmap.MapConf
	Transport http.RoundTripper // optional; nil uses http.DefaultTransport
	Obs       *obs.Registry     // optional; nil disables observability recording
}

type Crawler struct {
	conf       CrawlConf
	mainClient *Client
}

// NewCrawler allocates a Crawler with a pre-configured client
func NewCrawler(conf CrawlConf) *Crawler {
	cl := NewClient(conf.Upstream, conf.Transport)
	cl.SetObs(conf.Obs)
	return &Crawler{
		conf:       conf,
		mainClient: cl,
	}
}

// Fetch() crawl upstream map tree with a recursion limit.
// TODO: handle errors or unavailable maps
func (cr *Crawler) Fetch(ctx context.Context, startPath string, limit int) (
	chMap chan *mfmap.MfMap,
	chPicto chan mfmap.Picto,
) {
	chMap = make(chan (*mfmap.MfMap))
	chPicto = make(chan (mfmap.Picto))

	go func() {
		// closing channel signals a crawler exit and terminates server
		defer func() {
			close(chMap)
			close(chPicto)
		}()

		type QueueItem struct {
			path   string
			parent string
		}

		var (
			cnt      int
			wgPictos sync.WaitGroup
			queue    = []QueueItem{{startPath, ""}}
		)

		for {
			// stop when queue is empty, max count reached, or context expired
			i := len(queue) - 1
			if ((limit > 0) && (cnt >= limit)) || i < 0 {
				break
			}
			if err := ctx.Err(); err != nil {
				slog.Warn("fetch context expired", "startPath", startPath, "err", err)
				cr.recordCrawlError(startPath, err)
				break
			}
			cnt++
			// pop next map from queue
			next := queue[i]
			queue = queue[0:i]
			m, err := cr.getMap(ctx, next.path)
			if err != nil {
				slog.Error("getMap error", "path", next.path, "err", err)
				cr.recordMapFailed(next.path, err)
				continue
			}
			// add parent path
			m.Parent = next.parent

			// enqueue children maps
			for _, sz := range m.Data.Subzones {
				queue = append(queue, QueueItem{sz.Path, m.Path()})
			}
			// donwload pictos
			// cache will avoid multiple downloads of same a picto
			cr.fetchPictos(ctx, m.Pictos, &wgPictos, chPicto)

			// send map and drop pointer because ownership is transferred
			chMap <- m
			m = nil
		}
		// wait pictos completion, then close the channels (deferred)
		wgPictos.Wait()
	}()
	// both channels are closed on goroutine termination (deferred)
	return chMap, chPicto
}

// getMap gets https://mf.com/zone html page and related data like
// svg map, pictos, forecasts and list of subzones
// related data is stored into MfMap fields
func (cr *Crawler) getMap(ctx context.Context, path string) (*mfmap.MfMap, error) {
	slog.Info("getMap", "path", path)

	// Clear any previous token so the HTML page request goes out unauthenticated.
	// The HTML endpoint is public and its Set-Cookie response re-mints a fresh
	// mfsession token for the subsequent authenticated API calls. This avoids
	// stale-token loops on long-running instances when upstream expires sessions.
	cr.mainClient.token.Set("")
	body, err := cr.mainClient.Get(ctx, path, CacheDisabled)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	// allocate a MfMap and initialize with received content
	m := &mfmap.MfMap{
		OriginalPath: path,
		Conf:         cr.conf.MapConf,
	}
	err = m.ParseHtml(body)
	if err != nil {
		return nil, err
	}

	// apiClient is a closure returning a preconfigured api client
	apiClient := func() (*Client, error) {
		apiBaseUrl, err := urls.ApiUrl(m.Data, "", nil)
		if err != nil {
			return nil, err
		}
		cl := NewClient(apiBaseUrl.String(), cr.conf.Transport)
		cl.SetObs(cr.conf.Obs)
		cl.token.Set(cr.mainClient.token.Get())
		cl.noSessionCookie = true // api server do not send auth tokens so dont expect any
		return cl, nil
	}

	// subqueries to retreive SVG, geographical subzones and actual forecasts
	if err = cr.getAsset(ctx, func() (*url.URL, error) { return urls.SvgUrl(m.Conf.Upstream, m.Data) }, m.ParseSvgMap, nil); err != nil {
		return nil, err
	}
	if err = cr.getAsset(ctx, func() (*url.URL, error) { return urls.GeographyUrl(m.Conf.Upstream, m.Data) }, m.ParseGeography, nil); err != nil {
		return nil, err
	}
	if err = cr.getAsset(ctx, func() (*url.URL, error) { return urls.ForecastUrl(m.Data) }, m.ParseMultiforecast, apiClient); err != nil {
		return nil, err
	}
	m.Schedule.MarkUpdate() // record update time
	return m, nil
}

// getAsset downloads a map asset and feeds result into MfMap via parser
func (cr *Crawler) getAsset(
	ctx context.Context,
	urlGetter func() (*url.URL, error), // closure (over a mfmap.MfMap) returning asset url
	parser func(io.Reader) error, // closure (over a mfmap.MfMap) parsing the content
	clientGetter func() (*Client, error),
) error {
	u, err := urlGetter()
	if err != nil {
		return err
	}
	cl := cr.mainClient
	if clientGetter != nil {
		cl, err = clientGetter()
		if err != nil {
			return err
		}
	}
	body, err := cl.Get(ctx, u.String(), CacheDefault)
	if err != nil {
		return err
	}
	defer body.Close()
	err = parser(body)
	if err != nil {
		return err
	}
	return nil
}

// fetchPictos retrieves pictos from upstream
// cr.mainCient has a cache to avoid multiple downloads
func (cr *Crawler) fetchPictos(ctx context.Context, names []string, wg *sync.WaitGroup, out chan<- mfmap.Picto) {
	wg.Add(1)
	go func() {
		for _, name := range names {
			p, err := cr.getPicto(ctx, name)
			if err != nil {
				slog.Error("getPicto error", "name", name, "err", err)
				cr.recordPictoFailed(name, err)
				continue
			}
			out <- mfmap.Picto{Name: name, Img: p}
		}
		wg.Done()
	}()
}

func (cr *Crawler) getPicto(ctx context.Context, name string) ([]byte, error) {
	url, err := cr.pictoURL(name)
	if err != nil {
		return nil, err
	}
	body, err := cr.mainClient.Get(ctx, url.String(), CacheDefault)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	b, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// nil-safe obs recorders keep tests and non-instrumented callers working
func (cr *Crawler) recordMapFailed(path string, err error) {
	if cr.conf.Obs != nil {
		cr.conf.Obs.RecordMapFailed(path, err)
	}
}

func (cr *Crawler) recordPictoFailed(name string, err error) {
	if cr.conf.Obs != nil {
		cr.conf.Obs.RecordPictoFailed(name, err)
	}
}

func (cr *Crawler) recordCrawlError(target string, err error) {
	if cr.conf.Obs != nil {
		cr.conf.Obs.RecordCrawlError(target, err)
	}
}

// exemple https://meteofrance.com/modules/custom/mf_tools_common_theme_public/svg/weather/p3j.svg
func (cr *Crawler) pictoURL(name string) (*url.URL, error) {
	elems := []string{
		cr.conf.Upstream,
		"modules",
		"custom",
		"mf_tools_common_theme_public",
		"svg",
		"weather",
		fmt.Sprintf("%s.svg", name),
	}
	u, err := url.Parse(strings.Join(elems, "/"))
	if err != nil {
		return nil, fmt.Errorf("pictoURL() error: %w", err)
	}
	return u, nil
}
