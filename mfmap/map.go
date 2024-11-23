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
	data   []byte
}

type JsonData struct {
	Path        PathType        `json:"path"`
	MapLayersV2 MapLayersV2Type `json:"mf_map_layers_v2"`
	ToolsCommon ToolsCommonType `json:"mf_tools_common"`
}

type PathType struct {
	BaseUrl    string `json:"baseUrl"`
	ScriptPath string `json:"scriptPath"`
}

type MapLayersV2Type struct {
	Nid         string `json:"nid"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	Taxonomy    string `json:"taxonomy"`
	PathAssets  string `json:"path_assets"`
	IdTechnique string `json:"field_id_technique"`
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

func (j *JsonData) ApiURL() string {
	// self.api_url="https://"+conf["site"]+"."+conf["base_url"]
	// # https://rpcache-aa.meteofrance.com/internet2018client/2.0

	conf := j.ToolsCommon.Config
	return fmt.Sprintf("https://%s.%s", conf.Site, conf.BaseUrl)
}

/*
// accessor
func (m *MfMap) Nom() string {
	return m.nom
}

// accessor
func (m *MfMap) Html() string {
	return m.html
}
*/

// accessor
func (m *MfMap) SetParent(parent *MfMap) {
	m.parent = parent
}

func NewFrom(r io.Reader) (*MfMap, error) {
	j, err := JsonFilter(r)
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(j)
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
func JsonFilter(src io.Reader) (io.Reader, error) {
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

func JsonParser(r io.Reader) (*JsonData, error) {
	var j JsonData
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(buf, &j)
	return &j, err
}
