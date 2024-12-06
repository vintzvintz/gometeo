package mfmap

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

type MfMap struct {
	Nom       string
	Parent    *MfMap
	Data      *MapData
	Forecasts *MultiforecastData
}

type MapData struct {
	Path     PathType               `json:"path"`
	Info     MapInfoType            `json:"mf_map_layers_v2"`
	Children []POIType              `json:"mf_map_layers_v2_children_poi"`
	Subzones map[string]SubzoneType `json:"mf_map_layers_v2_sub_zone"`
	Tools    ToolsType              `json:"mf_tools_common"`
}

type PathType struct {
	BaseUrl    string `json:"baseUrl"`
	ScriptPath string `json:"scriptPath"`
}

type MapInfoType struct {
	Nid         string `json:"nid"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	Taxonomy    string `json:"taxonomy"`
	PathAssets  string `json:"path_assets"`
	IdTechnique string `json:"field_id_technique"`
}

// lat et lgn are mixed type float / string
type stringFloat float64

type POIType struct {
	Title      string      `json:"title"`
	Lat        stringFloat `json:"lat"`
	Lng        stringFloat `json:"lng"`
	Path       string      `json:"path"`
	Insee      string      `json:"insee"`
	Taxonomy   string      `json:"taxonomy"`
	CodePostal string      `json:"code_postal"`
	Timezone   string      `json:"timezone"`
}

type SubzoneType struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

type ToolsType struct {
	Alias  string     `json:"alias"`
	Config ConfigType `json:"config"`
}

type ConfigType struct {
	BaseUrl string `json:"base_url"`
	Site    string `json:"site"`
	Domain  string `json:"domain"`
}

const (
	apiMultiforecast = "/multiforecast"
)

// for coordinates sanity checks
const (
	minLat = 35.0
	maxLat = 50.0
	minLng = -10.0
	maxLng = 15.0
)

func (m *MfMap) ParseHtml(html io.Reader) error {
	j, err := htmlFilter(html)
	if err != nil {
		return err
	}
	data, err := mapParser(j)
	if err != nil {
		return err
	}
	m.Data = data
	return nil
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
	return nil, fmt.Errorf("données JSON non trouvées")
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

// ApiURL builds API URL from "config" node
// typically : https://rpcache-aa.meteofrance.com/internet2018client/2.0/path
func (j *MapData) apiURL(path string, query *url.Values) (*url.URL, error) {
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

func (m *MfMap) forecastURL() (*url.URL, error) {
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
	query.Add("coords", strings.Join(ids, ","))

	return m.Data.apiURL(apiMultiforecast, &query)
}

// https://meteofrance.com/modules/custom/mf_map_layers_v2/maps/desktop/METROPOLE/geo_json/regin13-aggrege.json
func (m *MfMap) geographyURL() (*url.URL, error) {
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
func (m *MfMap) svgURL() (*url.URL, error) {
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
func pictoURL(picto string) (*url.URL, error) {
	elems := []string{
		"modules",
		"custom",
		"mf_tools_common_theme_public",
		"svg",
		"weather",
		fmt.Sprintf("%s.svg", strings.ToLower(picto)),
	}
	u, err := url.Parse("https://meteofrance.com/" + strings.Join(elems, "/"))
	if err != nil {
		return nil, fmt.Errorf("m.svgURL() error: %w", err)
	}
	return u, nil
}

// UnmarshalJSON unmarshals stringFloat fields
// lat and lng are received as a mix of float and strings
func (sf *stringFloat) UnmarshalJSON(b []byte) error {
	// convert the bytes into an interface
	// this will help us check the type of our value
	// if it is a string that can be converted into aa float we convert it
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
