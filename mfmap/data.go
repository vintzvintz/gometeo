package mfmap

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"

	sf "gometeo/stringfloat"
)

type (
	MapData struct {
		Info     MapInfo  `json:"mf_map_layers_v2"`
		Children []Poi    `json:"mf_map_layers_v2_children_poi"`
		Subzones Subzones `json:"mf_map_layers_v2_sub_zone"`
		Tools    MapTools `json:"mf_tools_common"`
	}

	MapInfo struct {
		Nid         string `json:"nid"`
		Name        string `json:"name"`
		Path        string `json:"path"`
		Taxonomy    string `json:"taxonomy"`
		PathAssets  string `json:"path_assets"`
		IdTechnique string `json:"field_id_technique"`
	}

	Poi struct {
		Title      string         `json:"title"`
		Lat        sf.StringFloat `json:"lat"`
		Lng        sf.StringFloat `json:"lng"`
		Path       string         `json:"path"`
		Insee      string         `json:"insee"`
		Taxonomy   string         `json:"taxonomy"`
		CodePostal string         `json:"code_postal"`
		Timezone   string         `json:"timezone"`
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

var szFilters = map[string]*regexp.Regexp{
	// "DEPARTEMENT": no subzones
	"REGION": regexp.MustCompile(`^DEPT[0-9][0-9AB]$`),
	"PAYS":   regexp.MustCompile(`^REGIN[0-9][0-9]$`),
}

// mapParser parses json data to initialise a MfMap data structure
func ParseData(r io.Reader) (*MapData, error) {
	var data MapData
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(buf, &data)
	if err != nil {
		return nil, err
	}

	// keep only selected subzones, excluding marine & montagne & outermer
	re, ok := szFilters[data.Info.Taxonomy]
	if ok {
		for id := range data.Subzones {
			if !re.MatchString(id) {
				delete(data.Subzones, id)	
			}
		}
	}
	return &data, nil
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
	return extractPath(m.Data.Info.Path)
}

var pathPattern = regexp.MustCompile(`^/previsions-meteo-france/(.+)/`)

func extractPath(mfPath string) string {
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
