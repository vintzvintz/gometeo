package crawl

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"gometeo/mfmap"
	"gometeo/testutils"
)

const minPictoCount = 20
const minPictoSize = 200 // bytes

func TestGetAllMaps(t *testing.T) {
	maps, pictos := getAllMapsTest(t)
	checkMapCollection(t, maps)
	checkPictos(t, pictos)
}

func TestGetMap(t *testing.T) {
	maps, pictos := getMapTest(t, "/")
	checkMap(t, maps)
	checkPictos(t, pictos)
}

func getAllMapsTest(t *testing.T) (MapCollection, PictoStore) {
	c := NewCrawler()
	pictos := PictoStore{}
	maps, err := c.GetAllMaps("/", pictos)
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

func checkMapCollection(t *testing.T, maps MapCollection) {
	for _, m := range maps {
		checkMap(t, m)
	}
}

func checkMap(t *testing.T, m *mfmap.MfMap) {
	// TODO add more relevant checks
	name := m.Name()
	t.Log(name)
}

// there should be at least xxx different pictos, with minimum size,
// and with a '<svg' tag
func checkPictos(t *testing.T, pictos PictoStore) {
	re := regexp.MustCompile("<svg")
	if len(pictos) < minPictoCount {
		t.Fatalf("found less than %d pictos ???", minPictoCount)
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
		path       string
		pic        string
		wantStatus int
	}{
		"notFound": {"/picto/wesh", "wesh", http.StatusNotFound},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.path, nil)
			req.SetPathValue("pic", test.pic)
			testutils.RunHandler(t, hdl, req, test.wantStatus)
		})
	}
}
