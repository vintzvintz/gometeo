package mfmap

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

type (
	MfMap struct {
		// data embedded in main html
		Data *MapData
		// other data called from main page
		Forecasts MultiforecastData
		SvgMap    []byte
		Geography *geoCollection

		// parent map are used to build breadcrumbs
		Parent *MfMap
	}

	MapData struct {
		//	Path     MapPath  `json:"path"`
		Info     MapInfo  `json:"mf_map_layers_v2"`
		Children []Poi    `json:"mf_map_layers_v2_children_poi"`
		Subzones Subzones `json:"mf_map_layers_v2_sub_zone"`
		Tools    MapTools `json:"mf_tools_common"`
	}

	MapPath struct {
		BaseUrl    string `json:"baseUrl"`
		ScriptPath string `json:"scriptPath"`
	}

	MapInfo struct {
		Nid         string `json:"nid"`
		Name        string `json:"name"`
		Path        string `json:"path"`
		Taxonomy    string `json:"taxonomy"`
		PathAssets  string `json:"path_assets"`
		IdTechnique string `json:"field_id_technique"`
	}

	// lat et lgn are mixed type float / string
	stringFloat float64

	Poi struct {
		Title      string      `json:"title"`
		Lat        stringFloat `json:"lat"`
		Lng        stringFloat `json:"lng"`
		Path       string      `json:"path"`
		Insee      string      `json:"insee"`
		Taxonomy   string      `json:"taxonomy"`
		CodePostal string      `json:"code_postal"`
		Timezone   string      `json:"timezone"`
	}

	Subzone struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}

	Subzones map[string]Subzone // key = IdTechnique

	MapTools struct {
		Alias  string    `json:"alias"`
		Config MapConfig `json:"config"`
	}

	MapConfig struct {
		BaseUrl string `json:"base_url"`
		Site    string `json:"site"`
		Domain  string `json:"domain"`
	}
)

const (
	apiMultiforecast = "/multiforecast"
)

var szFilters = map[string]*regexp.Regexp{
	//		"DEPARTEMENT":   no subzones
	"REGION": regexp.MustCompile(`^DEPT[0-9][0-9AB]$`),
	"PAYS":   regexp.MustCompile(`^REGIN[0-9][0-9]$`),
}

func (m *MfMap) ParseHtml(html io.Reader) error {
	j, err := htmlFilter(html)
	if err != nil {
		return err
	}
	data, err := mapParser(j)
	if err != nil {
		return err
	}
	// keep only selected subzones, excluding marine & montagne & outermer
	data.Subzones.filterSubzones(data.Info.Taxonomy)
	m.Data = data
	return nil
}

func (m *MfMap) ParseMultiforecast(r io.Reader) error {
	fc, err := parseMfCollection(r)
	if err != nil {
		return err
	}
	m.Forecasts = fc.Features
	return nil
}

func (m *MfMap) ParseSvgMap(r io.Reader) error {
	r, err := cropSVG(r)
	if err != nil {
		return err
	}
	svg, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	m.SvgMap = svg
	return nil
}

func (m *MfMap) ParseGeography(r io.Reader) error {
	if m.Geography != nil {
		return fmt.Errorf("MfMap.Geography already populated")
	}
	geo, err := parseGeoCollection(r)
	if err != nil {
		return err
	}
	// keep only geographical subzones having a reference in map metadata
	geoFeats := make(geoFeatures, 0, len(geo.Features))
	for _, feat := range geo.Features {
		sz, ok := m.Data.Subzones[feat.Properties.Prop0.Cible]
		if !ok {
			continue
		}
		// add subzone path (could also be done client-side)
		feat.Properties.CustomPath = extractPath(sz.Path)
		geoFeats = append(geoFeats, feat)
	}
	// check consistency
	got := len(geoFeats)
	want := len(m.Data.Subzones)
	if got != want {
		return fmt.Errorf("all subzones defined in map metadata should"+
			"have a geographical representation (got %d want %d)", got, want)
	}

	// add subzone paths



	geo.Features = geoFeats // cant simplify geo because m.Geography == nil
	m.Geography = geo
	return nil
}

func (sz Subzones) filterSubzones(taxonomy string) {

	re, ok := szFilters[taxonomy]
	if !ok {
		re = regexp.MustCompile(`$^`) // match nothing
	}
	if re == nil {
		return
	}
	for id := range sz {
		if !re.MatchString(id) {
			delete(sz, id)
			log.Printf("ignore subzone %s", id)
		}
	}
}

// isJsonTag detecte l'élement contenant les donnnées drupal
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

// JsonFilter extracts json data from an html page
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

// mapParser parses json data to initialise a MfMap data structure
func mapParser(r io.Reader) (*MapData, error) {
	var j MapData
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(buf, &j)
	return &j, err
}

// PictoNames() return a list of all pictos used on the map
func (m *MfMap) PictoNames() []string {
	pictos := make([]string, 0)
	for _, feat := range (*m).Forecasts {
		for _, prop := range feat.Properties.Forecasts {
			pictos = append(pictos, prop.WeatherIcon, prop.WindIcon)
		}
		// dailies (long-term) forecasts have only weather icon, no wind icon
		for _, prop := range feat.Properties.Dailies {
			pictos = append(pictos, prop.WeatherIcon)
		}
	}
	// remove duplicates
	slices.Sort(pictos)
	pictos = slices.Compact(pictos)

	// remove empty string (in 1st position in sorted slice)
	if len(pictos) > 0 && pictos[0] == "" {
		pictos = pictos[1:]
	}
	return pictos
}

// ApiURL builds API URL from "config" node
// typically : https://rpcache-aa.meteofrance.com/internet2018client/2.0/path
func (j *MapData) ApiURL(path string, query *url.Values) (*url.URL, error) {
	conf := j.Tools.Config
	var querystring string
	if query != nil {
		querystring = "?" + query.Encode()
	}
	// build an url.URL on path with query parameters
	raw := fmt.Sprintf("https://%s.%s%s%s",
		conf.Site,
		conf.BaseUrl,
		path,
		querystring)
	return url.Parse(raw)
}

func (m *MfMap) ForecastURL() (*url.URL, error) {
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

	return m.Data.ApiURL(apiMultiforecast, &query)
}

// https://meteofrance.com/modules/custom/mf_map_layers_v2/maps/desktop/METROPOLE/geo_json/regin13-aggrege.json
func (m *MfMap) GeographyURL() (*url.URL, error) {
	elems := []string{
		"modules",
		"custom",
		"mf_map_layers_v2",
		"maps",
		"desktop",
		m.Data.Info.PathAssets,
		"geo_json",
		strings.ToLower(m.Data.Info.IdTechnique) + "-aggrege.json",
	}
	u, err := url.Parse("https://meteofrance.com/" + strings.Join(elems, "/"))
	if err != nil {
		return nil, fmt.Errorf("m.geographyURL() error: %w", err)
	}
	return u, nil
}

// https://meteofrance.com/modules/custom/mf_map_layers_v2/maps/desktop/METROPOLE/pays007.svg
func (m *MfMap) SvgURL() (*url.URL, error) {
	elems := []string{
		"modules",
		"custom",
		"mf_map_layers_v2",
		"maps",
		"desktop",
		m.Data.Info.PathAssets,
		fmt.Sprintf("%s.svg", strings.ToLower(m.Data.Info.IdTechnique)),
	}
	u, err := url.Parse("https://meteofrance.com/" + strings.Join(elems, "/"))
	if err != nil {
		return nil, fmt.Errorf("m.svgURL() error: %w", err)
	}
	return u, nil
}

// https://meteofrance.com/modules/custom/mf_tools_common_theme_public/svg/weather/p3j.svg
func PictoURL(picto string) (*url.URL, error) {
	elems := []string{
		"modules",
		"custom",
		"mf_tools_common_theme_public",
		"svg",
		"weather",
		fmt.Sprintf("%s.svg", picto),
	}
	u, err := url.Parse("https://meteofrance.com/" + strings.Join(elems, "/"))
	if err != nil {
		return nil, fmt.Errorf("m.pictoURL() error: %w", err)
	}
	return u, nil
}

// UnmarshalJSON unmarshals stringFloat fields
// lat and lng have mixed float and string types sometimes
func (sf *stringFloat) UnmarshalJSON(b []byte) error {
	// convert the bytes into an interface
	// this will help us check the type of our value
	// if it is a string that can be converted into a float we convert it
	// otherwise we return an error
	var item interface{}
	if err := json.Unmarshal(b, &item); err != nil {
		return err
	}
	switch v := item.(type) {
	case float64:
		*sf = stringFloat(v)
	case int:
		*sf = stringFloat(float64(v))
	case string:
		// here convert the string into a float
		i, err := strconv.ParseFloat(v, 64)
		if err != nil {
			// the string might not be of float type
			// so return an error
			return err
		}
		*sf = stringFloat(i)
	}
	return nil
}

func (m *MfMap) Name() string {
	if m.Data == nil {
		return "undefined"
	}
	return m.Data.Info.Name
}


func (m *MfMap) Path() string {
	if m.Data == nil {
		return "undefined"
	}
	// cas particulier pour la page d'acceuil
	if m.Data.Info.IdTechnique == "PAYS007" {
		return "france"
	}
	return extractPath( m.Data.Info.Path )
	/*
	match := pathPattern.FindStringSubmatch(m.Data.Info.Path)
	if (match != nil) && (len(match) == 2) {
		return match[1]
	}
	return ""
	*/
}

var pathPattern = regexp.MustCompile(`^/previsions-meteo-france/(.+)/`)

func extractPath(mfPath string) string{
	match := pathPattern.FindStringSubmatch(mfPath)
	if (match != nil) && (len(match) == 2) {
		return match[1]
	}
	return ""
}

func (sz *Subzones) UnmarshalJSON(b []byte) error {
	*sz = make(Subzones)
	// try unmarshalling to a map[string]Subzone
	// unmarshalling to same type (*Subzones) is infinite recursion
	tmp := make(map[string]Subzone)
	err := json.Unmarshal(b, &tmp)
	if err != nil {
		// retry to parse as an empty array instead of a map
		typeError, ok := err.(*json.UnmarshalTypeError)
		if ok && typeError.Value == "array" {
			a := make([]string, 0)
			err = json.Unmarshal(b, &a)
			if (err != nil) || (len(a) > 0) {
				err = fmt.Errorf("Subzones json is meither an object (map[string]) nor an empty array, %w", err)
				return err
			}
			// empty array unmarshalled into empty map
			return nil
		}
		return err
	}
	*sz = tmp
	return nil
}
