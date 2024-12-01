package mfmap

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// mfCollection is root element of a multiforecast api response
type mfCollection struct {
	Type     string       `json:"type"`
	Features []*mfFeature `json:"features"`
}

type mfFeature struct {
	UpdateTime time.Time  `json:"update_time"`
	Type       string     `json:"type"`
	Geometry   Geometry   `json:"geometry"`
	Properties MfProperty `json:"properties"`
}

type MultiforecastData []*mfFeature

type Geometry struct {
	Type   string      `json:"type"`
	Coords Coordinates `json:"coordinates"`
}

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

const featureCollectionStr = "FeatureCollection"

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
		return nil, fmt.Errorf("error unmarshalling multiforecast data: %w", err)
	}
	if fc.Type != featureCollectionStr {
		return nil, fmt.Errorf("featureCollection.Type got %s want %s", fc.Type, featureCollectionStr)
	}
	return &fc, nil
}
