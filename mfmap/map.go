package mfmap

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html"
)

/*
class Mf_map:
  # Counterpart of MF forecast pages, with geographical, svg and forecast data
  def __init__(self, data, own_path, parent):
    conf = data["mf_tools_common"]["config"]
    self.api_url="https://"+conf["site"]+"."+conf["base_url"]    # https://rpcache-aa.meteofrance.com/internet2018client/2.0
    self.infos = data["mf_map_layers_v2"]
    self.pois = data["mf_map_layers_v2_children_poi"]
    self.subzones = data["mf_map_layers_v2_sub_zone"]
    self.own_path = own_path
    self.parent = parent
*/

type MfMap struct {
	nom    string
	parent *MfMap
	data   *JsonData
}

type JsonData struct {
	Path        PathType               `json:"path"`
	Layers      LayerType              `json:"mf_map_layers_v2"`
	Children    []ChildType            `json:"mf_map_layers_v2_children_poi"`
	SubZones    map[string]SubZoneType `json:"mf_map_layers_v2_sub_zone"`
	ToolsCommon ToolsCommonType        `json:"mf_tools_common"`
}

type PathType struct {
	BaseUrl    string `json:"baseUrl"`
	ScriptPath string `json:"scriptPath"`
}

type LayerType struct {
	Nid         string `json:"nid"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	Taxonomy    string `json:"taxonomy"`
	PathAssets  string `json:"path_assets"`
	IdTechnique string `json:"field_id_technique"`
}

type ChildType struct {
	Title      string  `json:"title"`
	Lat        float64 `json:"lat ,string"`
	Lng        float64 `json:"lng ,string"`
	Path       string  `json:"path"`
	Insee      string  `json:"insee"`
	Taxonomy   string  `json:"taxonomy"`
	CodePostal string  `json:"code_postal"`
	Timezone   string  `json:"timezone"`
}

type SubZoneType struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

type ToolsCommonType struct {
	Alias  string     `json:"alias"`
	Config ConfigType `json:"config"`
}

type ConfigType struct {
	BaseUrl string `json:"base_url"`
	Site    string `json:"site"`
	Domain  string `json:"domain"`
}

// accessor
func (m *MfMap) SetParent(parent *MfMap) {
	m.parent = parent
}

func NewFrom(html io.Reader) (*MfMap, error) {
	j, err := jsonFilter(html)
	if err != nil {
		return nil, err
	}
	data, err := jsonParser(j)
	if err != nil {
		return nil, err
	}
	return &MfMap{data: data}, nil
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
func jsonFilter(src io.Reader) (io.Reader, error) {
	z := html.NewTokenizer(src)
	var inJson bool
loop:
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken: // éventuellement io.EOF
			break loop
		case html.StartTagToken:
			inJson = isJsonTag(z.Token())
		case html.TextToken:
			if inJson {
				return strings.NewReader(z.Token().Data), nil
			}
		}
	}
	return nil, fmt.Errorf("données JSON non trouvées")
}

func jsonParser(r io.Reader) (*JsonData, error) {
	var j JsonData
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(buf, &j)
	return &j, err
}

// ApiURL builds API URL from "config" node
// typically : https://rpcache-aa.meteofrance.com/internet2018client/2.0
func (j *JsonData) ApiURL() string {
	conf := j.ToolsCommon.Config
	return fmt.Sprintf("https://%s.%s", conf.Site, conf.BaseUrl)
}
