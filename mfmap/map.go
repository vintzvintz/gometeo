package mfmap

import (
	_ "embed"
	"fmt"
	"io"
	"log"
	"net/url"
	"strings"
	"text/template"

	"golang.org/x/net/html"

	"gometeo/appconf"
	gj "gometeo/geojson"
)

// MfMap is the main in-memory storage type of this project.
// Holds all dynamic data
type MfMap struct {

	// original path used on the GET request sent upstream
	OriginalPath string

	// Data is the result of parsing upstream html main page
	Data *MapData

	Prevs     gj.PrevList  // = gj.BuildPrevs()
	Graphdata gj.Graphdata // =  gj.BuildChroniques()

	Pictos []string

	// SvgMap is the background image (viewport-cropped upstream image)
	SvgMap []byte

	// Geography are geographical boundaries of subzones
	Geography gj.GeoCollection

	// breadcrumb is built by recursive parent lookup in MapCollection
	Parent     string
	Breadcrumb Breadcrumbs

	// unexported - use concurrence-safe accessors instead
	stats atomicStats
}

type (
	BreadcrumbItem struct {
		Nom  string `json:"nom"`
		Path string `json:"path"`
	}

	Breadcrumbs []BreadcrumbItem
)

const ApiMultiforecast = "/multiforecast"

// TemplateData contains data for htmlTemplate.Execute()
type TemplateData struct {
	Description string
	Title       string
	Path        string
	VueJs       string
	CacheId     string
}

//go:embed template.html
var templateFile string

// htmlTemplate is a global html/template for html rendering
var htmlTemplate = template.Must(template.New("").Parse(templateFile))

// main html file
func (m *MfMap) WriteHtml(wr io.Writer) error {
	return htmlTemplate.Execute(wr, &TemplateData{
		Description: fmt.Sprintf("Météo pour la zone %s sur une page grande et unique", m.Data.Info.Name),
		Title:       fmt.Sprintf("Météo %s", m.Data.Info.Name),
		Path:        m.Path(),
		CacheId:     appconf.CacheId(),
		VueJs:       appconf.VueJs(),
	})
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
	m.Data = data
	return nil
}

// Merge() recovers pastDays of backlog for Prevs and Chroniques
func (m *MfMap) Merge(old *MfMap, dayMin, dayMax int) {
	// sanity check
	if (m.Name() != old.Name()) || (m.Path() != old.Path()) {
		log.Print("MfMap.Merge() : name or path mismatch")
		return
	}
	// merge maps and chroniques
	m.Prevs.Merge(old.Prevs, dayMin, dayMax)
	m.Graphdata.Merge(old.Graphdata, dayMin, dayMax)

	// copy stats
	m.stats.lastHit.Store( old.stats.lastHit.Load() )
	m.stats.hitCount.Store( old.stats.hitCount.Load() )
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

func (m *MfMap) ForecastUrl() (*url.URL, error) {
	// zone is described by a seqence of coordinates
	ids := make([]string, len(m.Data.Children))
	for i, poi := range m.Data.Children {
		ids[i] = poi.Insee
	}
	query := make(url.Values)
	query.Add("bbox", "")
	query.Add("begin_time", "")
	query.Add("end_time", "")
	query.Add("time", "")
	query.Add("instants", "morning,afternoon,evening,night")
	query.Add("liste_id", strings.Join(ids, ","))

	return m.ApiUrl(ApiMultiforecast, &query)
}

func (m *MfMap) ParseMultiforecast(r io.Reader) error {
	fc, err := gj.ParseMultiforecast(r)
	if err != nil {
		return err
	}
	prevs, err := fc.Features.BuildPrevs()
	if err != nil {
		return err
	}
	graphdata, err := fc.Features.BuildChroniques()
	if err != nil {
		return err
	}
	m.Prevs = prevs
	m.Graphdata = graphdata
	m.Pictos = fc.Features.PictoNames()
	return nil
}

// https://meteofrance.com/modules/custom/mf_map_layers_v2/maps/desktop/METROPOLE/geo_json/regin13-aggrege.json
func (m *MfMap) GeographyUrl() (*url.URL, error) {
	elems := []string{
		appconf.UPSTREAM_ROOT,
		"modules",
		"custom",
		"mf_map_layers_v2",
		"maps",
		"desktop",
		m.Data.Info.PathAssets,
		"geo_json",
		strings.ToLower(m.Data.Info.IdTechnique) + "-aggrege.json",
	}
	u, err := url.Parse(strings.Join(elems, "/"))
	if err != nil {
		return nil, fmt.Errorf("m.GeographyUrl() error: %w", err)
	}
	return u, nil
}

// ParseGeography() parses a response from "geography" api endpoint.
// Calls geojson.ParseGeography() with valid subzones list.
func (m *MfMap) ParseGeography(r io.Reader) error {
	if len(m.Geography.Features) != 0 {
		return fmt.Errorf("MfMap.Geography already populated")
	}
	subzones := make(map[string]string)
	for sz := range m.Data.Subzones {
		path := extractPath(m.Data.Subzones[sz].Path)
		if path == "" {
			continue
		}
		subzones[sz] = path
	}
	gc, err := gj.ParseGeography(r, subzones)
	if err != nil {
		return err
	}
	if gc != nil {
		m.Geography = *gc
	}
	return nil
}
