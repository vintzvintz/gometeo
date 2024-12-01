package mfmap

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// mfCollection is root element of a multiforecast api response
type mfCollection struct {
	Type     FeatureCollectionType `json:"type"`
	Features MultiforecastData     `json:"features"`
}

// FeatureCollectionType is checked to be equal to "FeatureCollection"
// in custom UnmarshalJSON() method
type FeatureCollectionType string

type MultiforecastData []*mfFeature

type mfFeature struct {
	UpdateTime time.Time   `json:"update_time"`
	Type       FeatureType `json:"type"`
	Geometry   Geometry    `json:"geometry"`
	Properties MfProperty  `json:"properties"`
}

// FeatureCollectionType is checked to be equal to "Feature"
// in custom UnmarshalJSON() method
type FeatureType string

type Geometry struct {
	Type   PointType   `json:"type"`
	Coords Coordinates `json:"coordinates"`
}

type PointType string

type Coordinates [2]float64

type MfProperty struct {
	Name      string     `json:"name"`
	Country   string     `json:"country"`
	Dept      string     `json:"french_department"`
	Timezone  string     `json:"timezone"`
	Insee     string     `json:"insee"`
	Altitude  int        `json:"altitude"`
	Forecasts []Forecast `json:"forecast"`
	Dailies   []Daily    `json:"daily_forecast"`
}

type Forecast struct {
	Moment        string    `json:"moment_day"`
	Time          time.Time `json:"time"`
	T             float64   `json:"T"`
	TWindchill    float64   `json:"T_windchill"`
	WindSpeed     int       `json:"wind_speed"`
	WindSpeedGust int       `json:"wind_speed_gust"`
	WindDirection int       `json:"wind_direction"`
	WindIcon      string    `json:"wind_icon"`
	Iso0          int       `json:"iso0"`
	CloudCover    int       `json:"total_cloud_cover"`
	WeatherIcon   string    `json:"weather_icon"`
	WeatherDesc   string    `json:"weather_description"`
	Humidity      int       `json:"relative_humidity"`
	Pression      float64   `json:"P_sea"`
	Confiance     int       `json:"weather_confidence_index"`
}

type Daily struct {
	Time             time.Time `json:"time"`
	T_min            float64   `json:"T_min"`
	T_max            float64   `json:"T_max"`
	HumidityMin      int       `json:"relative_humidity_min"`
	HumidityMax      int       `json:"relative_humidity_max"`
	Uv               int       `json:"uv_index"`
	DailyWeatherIcon string    `json:"daily_weather_icon"`
	DailyWeatherDesc string    `json:"daily_weather_description"`
}

const (
	featureCollectionStr = "FeatureCollection"
	featureStr           = "Feature"
	pointStr             = "Point"
)

func unmarshalConstantString(b []byte, want string, name string) (string, error) {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return "", fmt.Errorf("%s unmarshal error: %w", name, err)
	}
	if s != want {
		return "", fmt.Errorf("%s is '%s' want '%s'", name, s, want)
	}
	return s, nil
}

func (fct *FeatureCollectionType) UnmarshalJSON(b []byte) error {
	s, err := unmarshalConstantString(b, featureCollectionStr, "FeatureCollection.Type")
	if err != nil {
		return err
	}
	*fct = FeatureCollectionType(s)
	return nil
}

func (fct *FeatureType) UnmarshalJSON(b []byte) error {
	s, err := unmarshalConstantString(b, featureStr, "Feature.Type")
	if err != nil {
		return err
	}
	*fct = FeatureType(s)
	return nil
}

func (fct *PointType) UnmarshalJSON(b []byte) error {
	s, err := unmarshalConstantString(b, pointStr, "Point.Type")
	if err != nil {
		return err
	}
	*fct = PointType(s)
	return nil
}

func parseMultiforecast(r io.Reader) (MultiforecastData, error) {
	fc, err := parseMfCollection(r)
	if err != nil {
		return nil, err
	}
	return fc.Features, nil
}

func parseMfCollection(r io.Reader) (*mfCollection, error) {

	var fc mfCollection
	j, err := io.ReadAll(r)

	if err != nil {
		return nil, fmt.Errorf("could not read multiforecast data: %w", err)
	}
	err = json.Unmarshal(j, &fc)
	if err != nil {
		return nil, fmt.Errorf("multiforecast data unmarshalling: %w", err)
	}
	return &fc, nil
}
