package crawl

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"gometeo/mfmap"
	"gometeo/testutils"
)

const minPictoCount = 10
const minPictoSize = 200 // bytes

func TestPictoUrl(t *testing.T) {
	u, err := pictoURL("test")
	if err != nil {
		t.Fatal(err)
	}
	got := u.String()
	want := "https://meteofrance.com/modules/custom/mf_tools_common_theme_public/svg/weather/test.svg"
	if got != want {
		t.Errorf("svgPicto() got '%s' want '%s'", got, want)
	}
}

func TestGetAllMaps(t *testing.T) {
	var cnt int = 10
	content := getAllMapsTest(t, cnt)
	checkMeteoContent(t, content, cnt)
}

func TestGetMap(t *testing.T) {
	maps, pictos := getMapTest(t, "/")
	checkMap(t, maps)
	checkPictos(t, pictos)
}

func getAllMapsTest(t *testing.T, limit int) *MeteoContent {
	c := NewCrawler()
	content, err := c.FetchAll("/", limit)
	//maps, err := c.GetAllMaps("/previsions-meteo-france/jura/39", nil)
	if err != nil {
		t.Fatalf("GetAllMaps() error: %s", err)
	}
	return content
}

func getMapTest(t *testing.T, path string) (*mfmap.MfMap, PictoStore) {
	cr := NewCrawler()
	m, err := cr.getMap(path)
	if err != nil {
		t.Fatalf("getmap('%s') error: %s", path, err)
	}
	pictos := PictoStore{}
	pictos.Update(m.PictoNames(), cr)
	return m, pictos
}

func checkMeteoContent(t *testing.T, c *MeteoContent, wantN int) {
	if len(c.maps) != wantN {
		t.Errorf("donwload %d maps, want %d ", len(c.maps), wantN)
	}
	checkPictos(t, c.pictos)
	for _, m := range c.maps {
		checkMap(t, m)
	}
}

func checkMap(t *testing.T, m *mfmap.MfMap) {
	if m.Data == nil {
		t.Errorf("MfMap field m.Data is nil")
	}
	if m.Forecasts == nil {
		t.Errorf("MfMap field m.Forecasts is nil")
	}
	if m.Geography == nil {
		t.Errorf("MfMap field m.Geography is nil")
	}
	if m.SvgMap == nil {
		t.Errorf("MfMap field m.SvgMap is nil")
	}
}

// there should be at least xxx different pictos, with minimum size,
// and with a '<svg' tag
func checkPictos(t *testing.T, pictos PictoStore) {
	re := regexp.MustCompile("<svg")
	if len(pictos) < minPictoCount {
		t.Fatalf("found %d pictos, expected at least %d", len(pictos), minPictoCount)
	}
	for p := range pictos {
		svg := pictos[p]
		if !re.Match(svg) {
			t.Errorf("no <svg> tag in picto '%s'", p)
		}
		if len(svg) < minPictoSize {
			t.Errorf("picto %s size too small ( %d < %d )", p, len(svg), minPictoSize)
		}
	}
}

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
