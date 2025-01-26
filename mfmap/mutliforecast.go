package mfmap

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"
)

// mfCollection is root element of a multiforecast api response
type mfCollection struct {
	Type     FeatureCollectionType `json:"type"`
	Features MultiforecastData     `json:"features"`
}

type MultiforecastData []mfFeature

type mfFeature struct {
	UpdateTime time.Time    `json:"update_time"`
	Type       FeatureType  `json:"type"`
	Geometry   MfGeometry   `json:"geometry"`
	Properties MfProperties `json:"properties"`
}

type MfGeometry struct {
	Type   PointType   `json:"type"`
	Coords Coordinates `json:"coordinates"`
}

type Coordinates struct {
	Lat, Lng float64
}

type MfProperties struct {
	Name      string     `json:"name"`
	Country   countryFr  `json:"country"`
	Dept      string     `json:"french_department"`
	Timezone  tzParis    `json:"timezone"`
	Insee     codeInsee  `json:"insee"`
	Altitude  int        `json:"altitude"`
	Forecasts []Forecast `json:"forecast"`
	Dailies   []Daily    `json:"daily_forecast"`
}

type Forecast struct {
	Moment        MomentName `json:"moment_day"`
	Time          time.Time  `json:"time"`
	T             float64    `json:"T"`
	TWindchill    float64    `json:"T_windchill"`
	WindSpeed     int        `json:"wind_speed"`
	WindSpeedGust int        `json:"wind_speed_gust"`
	WindDirection int        `json:"wind_direction"`
	WindIcon      string     `json:"wind_icon"`
	Iso0          int        `json:"iso0"`
	CloudCover    int        `json:"total_cloud_cover"`
	WeatherIcon   string     `json:"weather_icon"`
	WeatherDesc   string     `json:"weather_description"`
	Hrel          int        `json:"relative_humidity"`
	Pression      float64    `json:"P_sea"`
	Confiance     int        `json:"weather_confidence_index"`
	
	// Calculated field
	LongTerme bool `json:"long_terme"`
}

type Daily struct {
	Time        time.Time `json:"time"`
	Tmin        float64   `json:"T_min"`
	Tmax        float64   `json:"T_max"`
	Hmin        int       `json:"relative_humidity_min"`
	Hmax        int       `json:"relative_humidity_max"`
	Uv          int       `json:"uv_index"`
	WeatherIcon string    `json:"daily_weather_icon"`
	WeatherDesc string    `json:"daily_weather_description"`

	// calculated field sent to client
	//LongTerme bool `json:"long_terme"`
}

// custom types with runtime validation on unmarshalled data
type (
	FeatureCollectionType string
	FeatureType           string
	PointType             string
	tzParis               string
	countryFr             string
	MomentName            string
	codeInsee             string
)

var (
	featureCollectionStr = regexp.MustCompile(`FeatureCollection`)
	featureStr           = regexp.MustCompile(`Feature`)
	pointStr             = regexp.MustCompile(`Point`)
	tzStr                = regexp.MustCompile(`Europe/(Paris)|(Rome)|(Zurich)|(Madrid)|(Brussels)`)
	countryFrStr         = regexp.MustCompile(`FR - France`)
)

const (
	codeInseeMinLen = 6
)

const (
	morningStr   = "matin"
	afternoonStr = "après-midi"
	eveningStr   = "soirée"
	nightStr     = "nuit"
	dailyStr     = "daily"
)

// momentsStr is an alias for the 4 moments, not including 'daily"
var momentsStr = []MomentName{morningStr, afternoonStr, eveningStr, nightStr}

const (
	apiMultiforecast = "/multiforecast"
)

func (m *MfMap) ForecastURL() (*url.URL, error) {
	// zone is described by a seqence of coordinates
	ids := make([]string, len(m.Data.Children))
	for i, poi := range m.Data.Children {
		ids[i] = poi.Insee
	}
	query := make(url.Values)
	query.Add("bbox", "")
	query.Add("begin_time", "")
	query.Add("end_time", "")
	query.Add("time", "")
	query.Add("instants", "morning,afternoon,evening,night")
	query.Add("liste_id", strings.Join(ids, ","))

	return m.Data.ApiURL(apiMultiforecast, &query)
}

func (m *MfMap) ParseMultiforecast(r io.Reader) error {
	fc, err := parseMfCollection(r)
	if err != nil {
		return err
	}
	m.Forecasts = fc.Features
	return nil
}

// Unmarshall into a Forecast struct. Sets f.OnlyLT true
// if incoming json fields T or wind_speed are null
func (f *Forecast) UnmarshalJSON(data []byte) error {
	// unmarshall into a temp var of diffent type to avoid infinite recursion
	type RawForecast Forecast
	rf := RawForecast{}
	if err := json.Unmarshal(data, &rf); err != nil {
		return err
	}
	*f = Forecast(rf)

	// unmarshall sentinel fields as pointers to detect 'null' json values
	testNull := struct {
		Temp      *float64 `json:"T"`
		WindSpeed *float64 `json:"wind_speed"`
	}{}
	if err := json.Unmarshal(data, &testNull); err != nil {
		return err
	}
	// mark forecast as long-term only if basic data is mssing
	f.LongTerme = (testNull.Temp == nil) || (testNull.WindSpeed == nil)
	return nil
}

// Unmarshall into a Forecast struct. Sets f.OnlyLT true
// if incoming json fields T or wind_speed are null
/*func (d *Daily) UnmarshalJSON(data []byte) error {
	// unmarshall into a temp var of diffent type to avoid infinite recursion
	type RawDaily Daily
	rd := RawDaily{}
	if err := json.Unmarshal(data, &rd); err != nil {
		return err
	}
	*d = Daily(rd)
	// daily always displayed as "long-term"
	d.LongTerme = true
	return nil
}*/

func unmarshalStringValidate(b []byte, want *regexp.Regexp, name string) (string, error) {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return "", fmt.Errorf("%s unmarshal error: %w", name, err)
	}
	if !want.MatchString(s) {
		return "", fmt.Errorf("%s is '%s' want '%s'", name, s, want.String())
	}
	return s, nil
}

func (fct *FeatureCollectionType) UnmarshalJSON(b []byte) error {
	s, err := unmarshalStringValidate(b, featureCollectionStr, "FeatureCollection.Type")
	if err != nil {
		return err
	}
	*fct = FeatureCollectionType(s)
	return nil
}

func (fct *FeatureType) UnmarshalJSON(b []byte) error {
	s, err := unmarshalStringValidate(b, featureStr, "Feature.Type")
	if err != nil {
		return err
	}
	*fct = FeatureType(s)
	return nil
}

func (pt *PointType) UnmarshalJSON(b []byte) error {
	s, err := unmarshalStringValidate(b, pointStr, "Point.Type")
	if err != nil {
		return err
	}
	*pt = PointType(s)
	return nil
}

func (tz *tzParis) UnmarshalJSON(b []byte) error {
	s, err := unmarshalStringValidate(b, tzStr, "MfProperty.Timezone")
	if err != nil {
		return err
	}
	*tz = tzParis(s)
	return nil
}

func (ctry *countryFr) UnmarshalJSON(b []byte) error {
	s, err := unmarshalStringValidate(b, countryFrStr, "MfProperty.Country")
	if err != nil {
		return err
	}
	*ctry = countryFr(s)
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

func (code *codeInsee) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("code insee unmarshal error: %w", err)
	}
	if len(s) < codeInseeMinLen {
		return fmt.Errorf("code insee '%s' length=%d, expected >= %d bytes", s, len(s), codeInseeMinLen)
	}
	*code = codeInsee(s)
	return nil
}

func (m *MomentName) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("moment unmarshal error: %w", err)
	}
	allowedNames := []string{morningStr, afternoonStr, eveningStr, nightStr}
	for _, name := range allowedNames {
		if s == name {
			*m = MomentName(s)
			return nil
		}
	}
	return fmt.Errorf("moment '%s' not in known values  %v", s, allowedNames)
}

func parseMfCollection(r io.Reader) (*mfCollection, error) {
	var fc mfCollection
	j, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("could not read multiforecast data: %w", err)
	}
	err = json.Unmarshal(j, &fc)
	if err != nil {
		return nil, fmt.Errorf("invalid multiforecast: %w", err)
	}
	return &fc, nil
}

// PictoNames() return a list of all pictos used on the map
func (m *MfMap) PictoNames() []string {
	pictos := make([]string, 0)
	for _, feat := range m.Forecasts {
		for _, prop := range feat.Properties.Forecasts {
			pictos = append(pictos, prop.WeatherIcon, prop.WindIcon)
		}
		// dailies (long-term) forecasts have only weather icon, no wind icon
		for _, prop := range feat.Properties.Dailies {
			pictos = append(pictos, prop.WeatherIcon)
		}
	}
	// remove duplicates
	slices.Sort(pictos)
	pictos = slices.Compact(pictos)

	// remove empty string (in 1st position in sorted slice)
	if len(pictos) > 0 && pictos[0] == "" {
		pictos = pictos[1:]
	}
	return pictos
}
