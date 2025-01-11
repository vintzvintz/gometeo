package mfmap

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
)

type GeoCollection struct {
	Type     FeatureCollectionType `json:"type"`
	Bbox     Bbox                  `json:"bbox"`
	Features geoFeatures           `json:"features"`
}

type Bbox struct {
	LngW float64 `json:"w"`
	LngE float64 `json:"e"`
	LatN float64 `json:"n"`
	LatS float64 `json:"s"`
}

type geoFeatures []*geoFeature

type geoFeature struct {
	Bbox       Bbox        `json:"bbox"`
	Type       FeatureType `json:"type"`
	Properties GeoProperty `json:"properties"`
	Geometry   GeoGeometry `json:"geometry"`
}

type GeoProperty struct {
	Prop0 Prop0 `json:"prop0"`
	// Prop1 Prop1 `json:"prop1"`
	// Prop2 Prop2 `json:"prop2"`

	// subzones path is not provided in geography object
	// needs to be derived from prop0.cible (IdTechnique)
	CustomPath string `json:"customPath"`
}

type GeoGeometry struct {
	Type   PolygonType     `json:"type"`
	Coords [][]Coordinates `json:"coordinates"`
}

type Prop0 struct {
	Nom   string `json:"nom"`
	Cible string `json:"cible"`
	Paths Paths  `json:"paths"`
}

type Paths struct {
	Fr string `json:"fr"`
	En string `json:"en"`
	Es string `json:"es"`
}

type PolygonType string

// for coordinates sanity checks
const (
	minLat = 35.0
	maxLat = 55.0
	minLng = -12.0
	maxLng = 15.0
)

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
		// add subzone path (could also be done client-side but simpler here)
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

	geo.Features = geoFeats // cant simplify geo because m.Geography == nil
	m.Geography = geo
	return nil
}

var polygonStr = regexp.MustCompile("Polygon")

func (bbox *Bbox) UnmarshalJSON(b []byte) error {
	var a [4]float64
	if err := json.Unmarshal(b, &a); err != nil {
		return fmt.Errorf("bbox unmarshal error: %w. Want a [4]float64 array", err)
	}
	if err := checkLng(a[0]); err != nil {
		return err
	}
	bbox.LngW = a[0]
	if err := checkLat(a[1]); err != nil {
		return err
	}
	bbox.LatN = a[1]
	if err := checkLng(a[2]); err != nil {
		return err
	}
	bbox.LngE = a[2]
	if err := checkLat(a[3]); err != nil {
		return err
	}
	bbox.LatS = a[3]
	return nil
}

func (b Bbox) Crop() Bbox {
	return Bbox{
		LngW: b.LngW + cropPc.Left*(b.LngE-b.LngW),
		LatS: b.LatS + cropPc.Bottom*(b.LatN-b.LatS),
		LngE: b.LngE - cropPc.Right*(b.LngE-b.LngW),
		LatN: b.LatN - cropPc.Top*(b.LatN-b.LatS),
	}
}

func (pt *PolygonType) UnmarshalJSON(b []byte) error {
	s, err := unmarshalStringValidate(b, polygonStr, "GeoGeometry.Type")
	if err != nil {
		return err
	}
	*pt = PolygonType(s)
	return nil
}

func parseGeoCollection(r io.Reader) (*GeoCollection, error) {
	var gc GeoCollection
	j, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("could not read geography data: %w", err)
	}
	err = json.Unmarshal(j, &gc)
	if err != nil {
		return nil, fmt.Errorf("invalid geography: %w", err)
	}
	return &gc, nil
}

func checkLng(lng float64) error {
	if lng < minLng || lng > maxLng {
		return fmt.Errorf("longitude %f out of bounds [%f, %f]", lng, minLng, maxLng)
	}
	return nil
}

func checkLat(lat float64) error {
	if lat < minLat || lat > maxLat {
		return fmt.Errorf("latitude %f out of bounds [%f, %f]", lat, minLat, maxLat)
	}
	return nil
}
