package mfmap

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html"
)

// MfMap is the main in-memory storage type of this project.
// Holds all dynamic data
type MfMap struct {
	// Data is the result of parsing upstream html main page
	Data *MapData

	// Forecasts holds forecast data parsed from upstream json
	Forecasts MultiforecastData

	// SvgMap is the background image (viewport-cropped upstream image)
	SvgMap []byte

	// Geography are geographical boundaries of subzones
	Geography *GeoCollection

	// breadcrumb is built by recursive parent lookup in MapCollection
	Parent     string
	Breadcrumb Breadcrumb

	// unexported - use concurrence-safe accessors instead
	stats atomicStats
}

func (m *MfMap) ParseHtml(html io.Reader) error {
	j, err := htmlFilter(html)
	if err != nil {
		return err
	}
	data, err := ParseData(j)
	if err != nil {
		return err
	}
	// keep only selected subzones, excluding marine & montagne & outermer
	data.Subzones.filterSubzones(data.Info.Taxonomy)
	m.Data = data
	return nil
}

// htmFlilter extracts the json data part of an html page
func htmlFilter(src io.Reader) (io.Reader, error) {
	z := html.NewTokenizer(src)
	var inJson bool
loop:
	for {
		tt := z.Next()
		if tt == html.ErrorToken { // éventuellement io.EOF
			break loop
		}
		token := z.Token()
		switch tt {
		case html.StartTagToken:
			inJson = isJsonTag(token)
		case html.TextToken:
			if inJson {
				return strings.NewReader(token.Data), nil
			}
		}
	}
	return nil, fmt.Errorf("JSON data not found")
}

// isJsonTag detects DOM element holding drupal data
// <script type="application/json" data-drupal-selector="drupal-settings-json">
func isJsonTag(t html.Token) bool {
	isScript := (t.Type == html.StartTagToken) && (t.Data == "script")
	if !isScript {
		return false
	}
	var hasAttrType, hasDrupalAttr bool
	for _, a := range t.Attr {
		switch a.Key {
		case "type":
			hasAttrType = (a.Val == "application/json")
		case "data-drupal-selector":
			hasDrupalAttr = (a.Val == "drupal-settings-json")
		}
	}
	return isScript && hasAttrType && hasDrupalAttr
}
