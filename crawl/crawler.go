package crawl

import (
	"fmt"
	"io"
	"log"
	"net/url"
	"strings"
	"sync"

	"gometeo/mfmap"
)

const (
	httpsMeteofranceCom = "https://meteofrance.com"
	sessionCookie       = "mfsession"
)

type Crawler struct {
	mainClient *Client
}

// NewCrawler allocates a Crawler with a pre-configured client
func newCrawler() *Crawler {
	return &Crawler{
		mainClient: NewClient(httpsMeteofranceCom),
	}
}

type CrawlMode int

const (
	ModeOnce CrawlMode = iota
	ModeForever
)

// startCrawler returns a "self-updating" MeteoContent
func Start(path string, limit int, mode CrawlMode) (
	content *MeteoContent, done <-chan struct{}) {

	// concurrent use of http.Client is safe according to official documentation,
	// so we can share the same client for maps and pictos.
	cr := newCrawler()

	if (mode == ModeOnce)||(mode== ModeForever) {
		// direct pipe from crawler output channel to MeteoContent.Receive()
		content, done = cr.startOnce(path, limit)
	}
	if mode == ModeForever {
		// TODO update loop
	}
	if (content== nil) ||(done==nil) {
		panic(fmt.Errorf("crawl mode unknown : %d", int(mode)))		
	}
	return content, done

}

func (cr *Crawler) startOnce(startPath string, limit int) (
	content *MeteoContent, done <-chan struct{}) {
	// direct pipe from crawler output channel to MeteoContent.Receive()
	chMap, chPicto := cr.FetchOnce(startPath, limit)

	// returns a chan to signal when mapsChan is closed
	content = newContent()

	// bind crawler output channels to MeteoContent receive()
	contentDone := content.receive(chMap, chPicto)

	return content, contentDone
}

func (cr *Crawler) FetchOnce(startPath string, limit int) (
	chMap chan *mfmap.MfMap,
	chPicto chan *Picto,
) {
	chMap = make(chan (*mfmap.MfMap))
	chPicto = make(chan (*Picto))

	wgPictos := sync.WaitGroup{}
	go func() {
		// closing channel signals a crawler exit and terminates server
		defer func() {
			log.Printf("close chMap and chPicto")
			close(chMap)
			close(chPicto)
		}()
		var cnt int
		type QueueItem struct {
			path   string
			parent string
		}
		// root map has a nil parent
		queue := []QueueItem{{startPath, ""}}
		for {
			// stop when queue is empty or max count is reached
			i := len(queue) - 1
			if ((limit > 0) && (cnt >= limit)) || i < 0 {
				break
			}
			cnt++
			// pop next map from queue
			next := queue[i]
			queue = queue[0:i]
			m, err := cr.getMap(next.path)
			if err != nil {
				log.Printf("getMap(%s) error:%s", next.path, err)
				continue
			}
			// add parent path
			m.Parent = next.parent

			// enqueue children maps
			for _, sz := range m.Data.Subzones {
				queue = append(queue, QueueItem{sz.Path, m.Path()})
			}
			// send map
			chMap <- m

			// donwload pictos
			// cache will avoid multiple real downloads of same a picto
			cr.fetchPictos(m.PictoNames(), &wgPictos, chPicto)
		}

		// wait
		wgPictos.Wait()

		// signal goroutine termination
		log.Printf("crawl.FetchOnce('%s') exit", startPath)
	}()
	// both channels are closed on goroutine termination (deferred)
	return chMap, chPicto
}

// getMap gets https://mf.com/zone html page and related data like
// svg map, pictos, forecasts and list of subzones
// related data is stored into MfMap fields
func (cr *Crawler) getMap(path string) (*mfmap.MfMap, error) {
	log.Printf("getMap() '%s'", path)

	body, err := cr.mainClient.Get(path, CacheDisabled)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	// allocate a MfMap and initialize with received content
	m := &mfmap.MfMap{}
	err = m.ParseHtml(body)
	if err != nil {
		return nil, err
	}

	// prepare closure returning a preconfigured api client
	apiClient := func() (*Client, error) {
		apiBaseUrl, err := m.Data.ApiURL("", nil)
		if err != nil {
			return nil, err
		}
		cl := NewClient(apiBaseUrl.String())
		cl.authToken = cr.mainClient.authToken
		cl.noSessionCookie = true // api server do not send auth tokens so dont expect any
		return cl, nil
	}

	// subqueries to retreive SVG, geographical subzones and actual forecasts
	if err = cr.getAsset(m.SvgURL, m.ParseSvgMap, nil); err != nil {
		return nil, err
	}
	if err = cr.getAsset(m.GeographyURL, m.ParseGeography, nil); err != nil {
		return nil, err
	}
	if err = cr.getAsset(m.ForecastURL, m.ParseMultiforecast, apiClient); err != nil {
		return nil, err
	}
	m.LogUpdate() // record update time
	return m, nil
}

// getSvg() downloads SVG map and feed result into MfMap
func (cr *Crawler) getAsset(
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
		// cl = cr.createApiClient( cl // ientGetter()m.Data.ApiURL("", nil) )
	}
	body, err := cl.Get(u.String(), CacheDefault)
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
func (cr *Crawler) fetchPictos(names []string, wg *sync.WaitGroup, out chan *Picto) {
	wg.Add(1)
	go func() {
		for _, name := range names {
			p, err := cr.getPicto(name)
			if err != nil {
				log.Printf("getPicto(%s): %s", name, err)
				continue
			}
			out <- &Picto{name, p}
		}
		wg.Done()
	}()
}

func (cr *Crawler) getPicto(name string) ([]byte, error) {
	url, err := pictoURL(name)
	if err != nil {
		return nil, err
	}
	body, err := cr.mainClient.Get(url.String(), CacheDefault)
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

// exemple https://meteofrance.com/modules/custom/mf_tools_common_theme_public/svg/weather/p3j.svg
func pictoURL(name string) (*url.URL, error) {
	elems := []string{
		"modules",
		"custom",
		"mf_tools_common_theme_public",
		"svg",
		"weather",
		fmt.Sprintf("%s.svg", name),
	}
	u, err := url.Parse("https://meteofrance.com/" + strings.Join(elems, "/"))
	if err != nil {
		return nil, fmt.Errorf("pictoURL() error: %w", err)
	}
	return u, nil
}
