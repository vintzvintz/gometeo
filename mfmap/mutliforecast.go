package mfmap

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

type featureCollection struct {
	Type     string    `json:"type"`
	Features []feature `json:"features"`
}

type feature struct {
	UpdateTime time.Time `json:"update_time"`
	Type       string    `json:"type"`
}

type MultiforecastData []feature

const featureCollectionStr = "FeatureCollection"

func parseMultiforecast(r io.Reader) (MultiforecastData, error) {
	fc, err := parseFeatureCollection(r)
	if err != nil {
		return nil, err
	}
	return fc.Features, nil
}

func parseFeatureCollection(r io.Reader) (*featureCollection, error) {

	var fc featureCollection
	j, err := io.ReadAll(r)

	if err != nil {
		return nil, fmt.Errorf("could not read multiforecast data: %w", err)
	}
	err = json.Unmarshal(j, &fc)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling multiforecast data: %w", err)
	}
	if fc.Type != featureCollectionStr {
		return nil, fmt.Errorf("featureCollection.Type got %s want %s", fc.Type, featureCollectionStr)
	}
	return &fc, nil
}
