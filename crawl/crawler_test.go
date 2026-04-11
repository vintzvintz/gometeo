package crawl

import (
	"context"
	"regexp"
	"sync"
	"testing"
	"time"

	"gometeo/mfmap"
	"gometeo/mfmap/schedule"
)

var testCrawlConf = CrawlConf{
	Upstream: "https://meteofrance.com",
	MapConf: mfmap.MapConf{
		CacheId:  "testcache",
		VueJs:    "vue.esm-browser.dev.js",
		Upstream: "https://meteofrance.com",
		Rates: schedule.UpdateRates{
			HotDuration: 72 * time.Hour,
			HotMaxAge:   60 * time.Minute,
			ColdMaxAge:  240 * time.Minute,
		},
	},
}

const minPictosPerMap = 10 // temps, vent, uv....
const minPictoSize = 200   // bytes

func TestPictoUrl(t *testing.T) {
	cr := NewCrawler(testCrawlConf)
	u, err := cr.pictoURL("test")
	if err != nil {
		t.Fatal(err)
	}
	got := u.String()
	want := "https://meteofrance.com/modules/custom/mf_tools_common_theme_public/svg/weather/test.svg"
	if got != want {
		t.Errorf("svgPicto() got '%s' want '%s'", got, want)
	}
}

func TestFetch(t *testing.T) {
	var wantN int = 5
	maps, pictos := NewCrawler(testCrawlConf).Fetch(context.Background(), "/", wantN)

	var nbMaps, nbPics int
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		for m := range maps {
			nbMaps++
			checkMap(t, m)
		}
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		for p := range pictos {
			nbPics++
			checkPicto(t, p)
		}
		wg.Done()
	}()
	wg.Wait()

	if nbMaps != wantN {
		t.Errorf("download %d maps, want %d ", nbMaps, wantN)
	}
	wantPictos := minPictosPerMap * wantN
	if nbPics < wantPictos {
		t.Fatalf("found %d pictos, expected at least %d", nbPics, wantPictos)
	}
}

func TestGetMap(t *testing.T) {
	maps := getMapTest(t, "/")
	checkMap(t, maps)
}

func getMapTest(t *testing.T, path string) *mfmap.MfMap {
	cr := NewCrawler(testCrawlConf)
	m, err := cr.getMap(context.Background(), path)
	if err != nil {
		t.Fatalf("getmap('%s') error: %s", path, err)
	}
	return m
}

func checkMap(t *testing.T, m *mfmap.MfMap) {
	if m.Data == nil {
		t.Errorf("MfMap field m.Data is nil")
	}
	if m.Prevs == nil {
		t.Errorf("MfMap field m.Prevs is nil")
	}
	if m.Graphdata == nil {
		t.Errorf("MfMap field m.Graphdata is nil")
	}
	if len(m.Pictos) == 0 {
		t.Error("mfMap has no picto")
	}
	if m.Geography.Type != "FeatureCollection" {
		t.Errorf("MfMap.Geography has wrong type")
	}
	if m.SvgMap == nil {
		t.Errorf("MfMap field m.SvgMap is nil")
	}
}

var hasSvgTag *regexp.Regexp = regexp.MustCompile("<svg")

func checkPicto(t *testing.T, p mfmap.Picto) {
	//svg := p.Img
	if !hasSvgTag.Match(p.Img) {
		t.Errorf("no <svg> tag in picto '%s'", p.Name)
	}
	if len(p.Img) < minPictoSize {
		t.Errorf("picto %s too small size=%d minimum=%d", p.Name, len(p.Img), minPictoSize)
	}
}

// TODO Fix picothandler test
/*
func TestPictosHandler(t *testing.T) {
	_, pictos := getMapTest(t, "/")
	hdl := pictos.makePictosHandler()
	tests := map[string]struct {
		//path       string
		pic        string
		wantStatus int
	}{
		"notFound": {"wesh", http.StatusNotFound},
		"p3j":      {"p3j", http.StatusOK},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/pictos/"+test.pic, nil)
			req.SetPathValue("pic", test.pic)
			testutils.RunSvgHandler(t, hdl, req, test.wantStatus)
		})
	}
}
*/
