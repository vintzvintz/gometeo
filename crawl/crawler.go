package crawl

import (
	"fmt"
	"io"
	"log"
	"net/url"
	"strings"

	"gometeo/mfmap"
)

const (
	httpsMeteofranceCom = "https://meteofrance.com"
	sessionCookie       = "mfsession"
)

type Crawler struct {
	mainClient *MfClient
}

// NewCrawler allocates a Crawler with a pre-configured client
func NewCrawler() *Crawler {
	return &Crawler{
		mainClient: NewClient(httpsMeteofranceCom),
	}
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

	// get SVG, geographical subzones and actual forecasts
	if err = cr.getSvg(m); err != nil {
		return nil, err
	}
	if err = cr.getGeography(m); err != nil {
		return nil, err
	}
	if err = cr.getMultiforecast(m); err != nil {
		return nil, err
	} 
	return m, nil
}

// getSvg() downloads SVG map and feed result into MfMap
func (cr *Crawler) getSvg(m *mfmap.MfMap) error {
	u, err := m.SvgURL()
	if err != nil {
		return err
	}
	body, err := cr.mainClient.Get(u.String(), CacheDefault)
	if err != nil {
		return err
	}
	defer body.Close()
	err = m.ParseSvgMap(body)
	if err != nil {
		return err
	}
	return nil
}

func (cr *Crawler) getGeography(m *mfmap.MfMap) error {
	u, err := m.GeographyURL()
	if err != nil {
		return err
	}
	body, err := cr.mainClient.Get(u.String(), CacheDefault)
	if err != nil {
		return err
	}
	defer body.Close()
	err = m.ParseGeography(body)
	if err != nil {
		return err
	}
	return nil
}

func (cr *Crawler) getMultiforecast(m *mfmap.MfMap) error {
	// create a dedicated client for rpcache-aa host
	apiBaseUrl, err := m.Data.ApiURL("", nil)
	if err != nil {
		return err
	}
	api := NewClient(apiBaseUrl.String())
	api.authToken = cr.mainClient.authToken
	api.noSessionCookie = true // api server do not send auth tokens so dont expect any

	// get all forecasts available on the map
	u, err := m.ForecastURL()
	if err != nil {
		return err
	}
	body, err := api.Get(u.String(), CacheDisabled)
	if err != nil {
		return err
	}
	defer body.Close()
	err = m.ParseMultiforecast(body)
	if err != nil {
		return err
	}
	return nil
}

// FetchAll() fetches a map tree recursively, including pictos
// * startPath : where to start the tree walk ("/" is the 'root' page)
// * limit limits the number of maps downloaded
func (cr *Crawler) FetchAll(startPath string, limit int) (*MeteoContent, error) {
	var (
		cnt    int
		maps   = MapStore{}
		pictos = PictoStore{}
	)

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

		// pop queue and process next path
		next := queue[i]
		queue = queue[0:i]
		m, err := cr.getMap(next.path)
		if err != nil {
			return nil, err
		}
		// add parent
		m.Parent = next.parent

		// download pictos (only new ones)
		err = pictos.Update(m.PictoNames(), cr)
		if err != nil {
			return nil, err
		}

		// enqueue children maps
		for _, sz := range m.Data.Subzones {
			queue = append(queue, QueueItem{sz.Path, m.Path()})
		}

		// store current map in the collection
		maps[m.Path()] = m
	}
	// store maps and pictos in a MeteoContent
	mc := NewContent()
	mc.Update(maps, pictos)
	return mc, nil
}

func (cr *Crawler) Fetch(startPath string, limit int) <-chan *mfmap.MfMap {

	ch := make(chan (*mfmap.MfMap))

	go func() {
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

			// pop queue and process next path
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
			ch <- m
		}
		// signal goroutine termination
		log.Printf("crawl.Fetch('%s') exit", startPath)

		// closing channel will terminate server
		// we do not want that in "test/limited" mode
		if limit == 0 {
			close(ch)
		}
	}()
	return ch
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
