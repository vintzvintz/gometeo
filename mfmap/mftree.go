package mfmap

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
