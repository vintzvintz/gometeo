package crawl

import (
	"regexp"
	"testing"
)


const minPictoCount = 20
const minPictoSize = 200 // bytes

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

// there should be at least xxx different pictos, with minimum size,
// and with a '<svg' tag
func checkPictos(t *testing.T, c *Crawler) {
	re := regexp.MustCompile("<svg")
	pictos := c.Pictos()
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
