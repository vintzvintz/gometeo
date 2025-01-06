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

func TestGetAllMaps(t *testing.T) {
	var cnt int = 10
	maps, pictos := getAllMapsTest(t, cnt)

	if len(maps) != cnt {
		t.Errorf( "donwload %d maps, want %d ", len(maps), cnt)
	}
	checkMapCollection(t, maps)
	checkPictos(t, pictos)
}

func TestGetMap(t *testing.T) {
	maps, pictos := getMapTest(t, "/")
	checkMap(t, maps)
	checkPictos(t, pictos)
}

func getAllMapsTest(t *testing.T, limit int) (MapStore, PictoStore) {
	c := NewCrawler()
	pictos := PictoStore{}
	maps, err := c.FetchAll("/", pictos, limit)
	//maps, err := c.GetAllMaps("/previsions-meteo-france/jura/39", nil)
	if err != nil {
		t.Fatalf("GetAllMaps() error: %s", err)
	}
	return maps, pictos
}

func getMapTest(t *testing.T, path string) (*mfmap.MfMap, PictoStore) {
	c := NewCrawler()
	pictos := PictoStore{}
	m, err := c.GetMap(path, nil, pictos)
	if err != nil {
		t.Fatalf("Getmap('%s') error: %s", path, err)
	}
	return m, pictos
}

func checkMapCollection(t *testing.T, maps MapStore) {
	for _, m := range maps {
		checkMap(t, m)
	}
}

func checkMap(t *testing.T, m *mfmap.MfMap) {
	if( m.Data == nil ) {
		t.Errorf( "MfMap field m.Data is nil")
	}
	if( m.Forecasts == nil ) {
		t.Errorf( "MfMap field m.Forecasts is nil")
	}
	if( m.Geography == nil ) {
		t.Errorf( "MfMap field m.Geography is nil")
	}
	if( m.SvgMap == nil ) {
		t.Errorf( "MfMap field m.SvgMap is nil")
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
		"notFound": { "wesh", http.StatusNotFound},
		"p3j": {"p3j", http.StatusOK},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/pictos/"+test.pic, nil)
			req.SetPathValue("pic", test.pic)
			testutils.RunSvgHandler(t, hdl, req, test.wantStatus)
		})
	}
}
