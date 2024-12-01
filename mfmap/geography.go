package mfmap

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type geoCollection struct {
	Type     FeatureCollectionType `json:"type"`
	Bbox     *Bbox                 `json:"bbox"`
	Features []*geoFeature         `json:"features"`
}

type Bbox struct {
	A, B Coordinates
}

type geoFeature struct {
	Bbox       *Bbox       `json:"bbox"`
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

const polygonStr = "Polygon"

type cropParams struct {
	north, south, east, west float64
}

func (bbox *Bbox) UnmarshalJSON(b []byte) error {
	var a [4]float64
	if err := json.Unmarshal(b, &a); err != nil {
		return fmt.Errorf("bbox unmarshal error: %w. Want a [4]float64 array", err)
	}
	p1, err := NewCoordinates(a[0], a[1])
	if err != nil {
		return err
	}
	p2, err := NewCoordinates(a[2], a[3])
	if err != nil {
		return err
	}
	bbox.A, bbox.B = *p1, *p2
	return nil
}

func (pt *PolygonType) UnmarshalJSON(b []byte) error {
	s, err := unmarshalConstantString(b, polygonStr, "GeoGeometry.Type")
	if err != nil {
		return err
	}
	*pt = PolygonType(s)
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
		return nil, fmt.Errorf("invalid geography: %w", err)
	}
	return &gc, nil
}

func cropSVG(svg io.Reader, p cropParams) (io.Reader, error) {
	_ = p
	_ = svg
	return strings.NewReader("<wesh>weeesh</wesh>"), nil
}
