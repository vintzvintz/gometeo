package crawl

import (
	"testing"
)

/*
func TestGet(t *testing.T) {

	name := "test_data/racine.html"
	f, err := os.Open(name)
	if err != nil {
		t.Errorf("%s : %v", name, err)
	}

	// setup mock server
	srv := httptest.NewServer( http.HandlerFunc( func ( w http.ResponseWriter, req *http.Request ) {
		_, err := io.Copy(w, f)
		if err != nil {
			t.Error(err)
		}
	} ))
	defer srv.Close()

//	client := NewClient(nil)
//	client.Get( )
}
*/

func TestGetMap(t *testing.T) {
	path := "/"
	c := NewCrawler()
	m, err := c.GetMap(path, nil)
	if err != nil {
		t.Fatalf("Getmap('%s') error: %s", path, err)
	}
	name, err := m.Name()
	if err != nil {
		t.Fatalf("%s MfMmap.Name() error: %s", path, err)
	}

	// check pictos
	checkPictos(t, c)

	t.Log(name)
}

const minPictoCount = 5
const minPictoSize = 1024 // bytes

func checkPictos(t *testing.T, c *Crawler) {

	// there should be at least 5 different pictos, each > 1kb,
	// starting with a '<' character ( XML tag )

	pictos := c.Pictos()
	if len(pictos) < minPictoCount {
		t.Fatalf("found less than %d pictos ???", minPictoCount)
	}
	for p := range pictos {
		svg := pictos[p]
		if svg[0] != '<' {
			t.Errorf("picto %s content does not start with a '<' char", p)
		}
		if len(svg) < minPictoSize {
			t.Errorf("picto %s size too small ( %d < %d )", p, len(svg), minPictoSize)
		}
	}
}
