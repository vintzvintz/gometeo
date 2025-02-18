package geojson

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
)

type GeoCollection struct {
	Type     FeatureCollectionType `json:"type"`
	Bbox     Bbox                  `json:"bbox"`
	Features GeoFeatures           `json:"features"`
}

type Bbox struct {
	LngW float64 `json:"w"`
	LngE float64 `json:"e"`
	LatN float64 `json:"n"`
	LatS float64 `json:"s"`
}

type GeoFeatures []*geoFeature

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

type Coordinates struct {
	Lat, Lng float64
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

// ParseGeograpy parses a geojson object into a GeoCollection.
// subzones maps "geoFeature.feat.Properties.Prop0.Cible" to a CustomPath.
// Drops any Feature not referenced by subzones parameter
func ParseGeography(r io.Reader, subzones map[string]string) (*GeoCollection, error) {
	var gc GeoCollection
	j, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("could not read geography data: %w", err)
	}
	err = json.Unmarshal(j, &gc)
	if err != nil {
		return nil, fmt.Errorf("invalid geography: %w", err)
	}

	// keep only geographical subzones having a reference in map metadata
	geoFeats := make(GeoFeatures, 0, len(subzones))
	for _, feat := range gc.Features {
		cible := feat.Properties.Prop0.Cible
		path, ok := subzones[cible]
		if !ok {
			continue
		}
		// add subzone path (could also be done client-side but simpler here)
		feat.Properties.CustomPath = path // extractPath(sz.Path)
		geoFeats = append(geoFeats, feat)
	}
	gc.Features = geoFeats // cant simplify geo because m.Geography == nil
	return &gc, nil
}

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

func (b Bbox) Crop(left, right, top, bottom float64) Bbox {
	return Bbox{
		LngW: b.LngW + left*(b.LngE-b.LngW),
		LatS: b.LatS + bottom*(b.LatN-b.LatS),
		LngE: b.LngE - right*(b.LngE-b.LngW),
		LatN: b.LatN - top*(b.LatN-b.LatS),
	}
}

func (pt *PolygonType) UnmarshalJSON(b []byte) error {
	var polygonStr = regexp.MustCompile("Polygon")
	s, err := unmarshalStringValidate(b, polygonStr, "GeoGeometry.Type")
	if err != nil {
		return err
	}
	*pt = PolygonType(s)
	return nil
}


func (c *Coordinates) UnmarshalJSON(b []byte) error {
	// https://datatracker.ietf.org/doc/html/rfc7946#section-3.1.1
	var a [2]float64
	if err := json.Unmarshal(b, &a); err != nil {
		return fmt.Errorf("coordinates unmarshal error: %w. Want a [2]float64 array", err)
	}
	if err := checkLng(a[0]); err != nil {
		return err
	}
	if err := checkLat(a[1]); err != nil {
		return err
	}
	c.Lng, c.Lat = a[0], a[1]
	return nil
}

// MarshalJSON outputs lng/lat as [float, float]
// instead of default object {Lng:float, Lat:float}
// cf https://datatracker.ietf.org/doc/html/rfc7946#section-3.1.1
func (c *Coordinates) MarshalJSON() ([]byte, error) {
	return json.Marshal([]float64{c.Lng, c.Lat})
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
