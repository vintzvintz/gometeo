package mfmap

import (
	"encoding/json"
	"fmt"
	"io"
)

type geoCollection struct {
	Type     FeatureCollectionType `json:"type"`
	Bbox     Bbox                  `json:"bbox"`
	Features []*geoFeature         `json:"features"`
}

type Bbox struct {
	LngW, LngE, LatN, LatS float64
}

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

const polygonStr = "Polygon"

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
		LngW: b.LngW + cropPcLeft*(b.LngE-b.LngW),
		LatS: b.LatS + cropPcBottom*(b.LatN-b.LatS),
		LngE: b.LngE + cropPcRight*(b.LngE-b.LngW),
		LatN: b.LatN - cropPcTop*(b.LatN-b.LatS),
	}
}

func (pt *PolygonType) UnmarshalJSON(b []byte) error {
	s, err := unmarshalConstantString(b, polygonStr, "GeoGeometry.Type")
	if err != nil {
		return err
	}
	*pt = PolygonType(s)
	return nil
}

func parseGeoCollection(r io.Reader) (*geoCollection, error) {
	var gc geoCollection
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
