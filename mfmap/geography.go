package mfmap

import (
	"encoding/json"
	"fmt"
	"io"
)

type geoCollection struct {
	Type     string        `json:"type"`
	Bbox     *Bbox         `json:"bbox"`
//	Features []*geoFeature `json:"features"`
}

type Bbox struct {
	Lng1, Lat1 float64
	Lng2, Lat2 float64
}
/*
type geoFeature struct {
	Bbox       *Bbox       `json:"bbox"`
	Type       string      `json:"type"`
	Properties GeoProperty `json:"properties"`
	Geometry   GeoGeometry `json:"geometry"`
}

type GeoProperty struct {
	Prop0 Prop0 `json:"prop0"`
	// Prop1 Prop1 `json:"prop1"`
	// Prop2 Prop2 `json:"prop2"`
}

type GeoGeometry struct {
	Type   string          `json:"type"`
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
*/
func (bbox *Bbox) UnmarshalJSON(b []byte) error {
	var a [4]float64
	if err := json.Unmarshal(b, &a); err != nil {
		return fmt.Errorf("bbox unmarshal error: %w. Want a [4]float64 array", err)
	}
	bbox.Lng1, bbox.Lat1 = a[0], a[1]
	bbox.Lng2, bbox.Lat2 = a[2], a[3]
	return nil
}

func parseGeography(r io.Reader) (*geoCollection, error) {
	var gc geoCollection
	j, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("could not read geography data: %w", err)
	}
	err = json.Unmarshal(j, &gc)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling geography data: %w", err)
	}
	if gc.Type != featureCollectionStr {
		return nil, fmt.Errorf("featureCollection.Type got %s want %s", gc.Type, featureCollectionStr)
	}
	return &gc, nil
}
