package urls

import (
	"fmt"
	"net/url"
	"strings"

	"gometeo/mfmap"
)

const ApiMultiforecast = "/multiforecast"

// ApiUrl builds an API URL from MapData.Tools.Config
// typically: https://rpcache-aa.meteofrance.com/internet2018client/2.0/path
func ApiUrl(data *mfmap.MapData, path string, query *url.Values) (*url.URL, error) {
	conf := data.Tools.Config
	var querystring string
	if query != nil {
		querystring = "?" + query.Encode()
	}
	raw := fmt.Sprintf("https://%s.%s%s%s",
		conf.Site,
		conf.BaseUrl,
		path,
		querystring)
	return url.Parse(raw)
}

// ForecastUrl builds the multiforecast endpoint URL from MapData
func ForecastUrl(data *mfmap.MapData) (*url.URL, error) {
	ids := make([]string, len(data.Children))
	for i, poi := range data.Children {
		ids[i] = poi.Insee
	}
	query := make(url.Values)
	query.Add("bbox", "")
	query.Add("begin_time", "")
	query.Add("end_time", "")
	query.Add("time", "")
	query.Add("instants", "morning,afternoon,evening,night")
	query.Add("liste_id", strings.Join(ids, ","))

	return ApiUrl(data, ApiMultiforecast, &query)
}

// GeographyUrl builds the geography GeoJSON URL
// https://meteofrance.com/modules/custom/mf_map_layers_v2/maps/desktop/METROPOLE/geo_json/regin13-aggrege.json
func GeographyUrl(upstream string, data *mfmap.MapData) (*url.URL, error) {
	elems := []string{
		upstream,
		"modules",
		"custom",
		"mf_map_layers_v2",
		"maps",
		"desktop",
		data.Info.PathAssets,
		"geo_json",
		strings.ToLower(data.Info.IdTechnique) + "-aggrege.json",
	}
	u, err := url.Parse(strings.Join(elems, "/"))
	if err != nil {
		return nil, fmt.Errorf("GeographyUrl() error: %w", err)
	}
	return u, nil
}

// SvgUrl builds the SVG map URL
// https://meteofrance.com/modules/custom/mf_map_layers_v2/maps/desktop/METROPOLE/pays007.svg
func SvgUrl(upstream string, data *mfmap.MapData) (*url.URL, error) {
	elems := []string{
		upstream,
		"modules",
		"custom",
		"mf_map_layers_v2",
		"maps",
		"desktop",
		data.Info.PathAssets,
		fmt.Sprintf("%s.svg", strings.ToLower(data.Info.IdTechnique)),
	}
	u, err := url.Parse(strings.Join(elems, "/"))
	if err != nil {
		return nil, fmt.Errorf("SvgUrl() error: %w", err)
	}
	return u, nil
}
