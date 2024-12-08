package mfmap

import (
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"time"
)

// mfCollection is root element of a multiforecast api response
type mfCollection struct {
	Type     FeatureCollectionType `json:"type"`
	Features MultiforecastData     `json:"features"`
}

type MultiforecastData []mfFeature

type mfFeature struct {
	UpdateTime time.Time   `json:"update_time"`
	Type       FeatureType `json:"type"`
	Geometry   MfGeometry  `json:"geometry"`
	Properties MfProperty  `json:"properties"`
}

type MfGeometry struct {
	Type   PointType   `json:"type"`
	Coords Coordinates `json:"coordinates"`
}

type Coordinates struct {
	Lat, Lng float64
}

type MfProperty struct {
	Name      string     `json:"name"`
	Country   countryFr  `json:"country"`
	Dept      string     `json:"french_department"`
	Timezone  tzParis    `json:"timezone"`
	Insee     CodeInsee  `json:"insee"`
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
}

const (
	featureCollectionStr = "FeatureCollection"
	featureStr           = "Feature"
	pointStr             = "Point"
	tzStr                = "Europe/Paris"
	countryFrStr         = "FR - France"
	codeInseeMinLen      = 6
)

const (
	morningStr   = "matin"
	afternoonStr = "après-midi"
	eveningStr   = "soirée"
	nightStr     = "nuit"
)

// custom types with runtime checks on unmarshalled values
type FeatureCollectionType string
type FeatureType string
type PointType string
type tzParis string
type countryFr string
type MomentName string
type CodeInsee string

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

func (pt *PointType) UnmarshalJSON(b []byte) error {
	s, err := unmarshalConstantString(b, pointStr, "Point.Type")
	if err != nil {
		return err
	}
	*pt = PointType(s)
	return nil
}

func (tz *tzParis) UnmarshalJSON(b []byte) error {
	s, err := unmarshalConstantString(b, tzStr, "MfProperty.Timezone")
	if err != nil {
		return err
	}
	*tz = tzParis(s)
	return nil
}

func (ctry *countryFr) UnmarshalJSON(b []byte) error {
	s, err := unmarshalConstantString(b, countryFrStr, "MfProperty.Country")
	if err != nil {
		return err
	}
	*ctry = countryFr(s)
	return nil
}

func (c *Coordinates) UnmarshalJSON(b []byte) error {
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

func (code *CodeInsee) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("code insee unmarshal error: %w", err)
	}
	if len(s) < codeInseeMinLen {
		return fmt.Errorf("code insee '%s' length=%d, expected >= %d bytes", s, len(s), codeInseeMinLen)
	}
	*code = CodeInsee(s)
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

// pictoList() return a list of all pictos used on the map
func (mf MultiforecastData) pictoList() []string {
	pictos := make([]string, 0)
	for _, feat := range mf {
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
	// remove empty string
	if pictos[0] == "" {
		pictos = pictos[1:]
	}
	return pictos
}
/*
type PrevList map[Echeance]PrevsAtEch
*/
// Prevlist key is a composite type
type Echeance struct {
	Moment MomentName
	Day    time.Time // yyyy-mm-dd @ 00-00-00 UTC
}
/*
// all available forecasts for a single point of interest
type PrevsAtEch struct {
	Time    time.Time
	Updated time.Time
	Prevs   []PrevAtPoi
}

// forecast data for a single (poi, date) point
type PrevAtPoi struct {
	Title  string
	Coords Coordinates
	Short  *Forecast // empty or zero values after 10 days
	Daily  *Daily
}
*/

/*
	func (mf MultiforecastData) ByEcheance() PrevList {
		pl := make(PrevList)

		// iterate over POIs ( known as "Features" in json data )
		for _, feat := range mf {

			// process short-term forecasts first
			for _, short := range feat.Properties.Forecasts {

				// build an Echeance to use as PrevList key
				e := Echeance{
					Moment: short.Moment,
					Day: time.Date(
						short.Time.Year(),
						short.Time.Month(),
						short.Time.Day(),
						0, 0, 0, 0, // hour, min, sec, nano
						time.UTC),
				}

				// get prevsions@echeance struct
				pae, ok := pl[e]
				if !ok {
					// create Prev@Ech slice if it does not already exist
					pae = PrevsAtEch{
						Time:    short.Time,
						Updated: feat.UpdateTime,
						Prevs:   []PrevAtPoi{},
					}
				} else {
					// warns if echeances are not unique for different POIs on a same day+moment key
					if pae.Time != short.Time {
						log.Default().Printf("Inconsistent times for [%s] '%t' != '%t'",
							e, pae.Time, short.Time,
						)
					}
				}

				// get daily prev for the day/poi

				// wrap forecast and daily in a struct
				p := PrevAtPoi{
					Title:    feat.Properties.Name,
					Coords:   feat.Geometry.Coords,
					Forecast: short,
				}
				pae.Prevs = append(pae.Prevs, p)

			}

			// add daily data to each "short-term" forecast
			for _, daily := range feat.Properties.Dailies {
				dailies[daily.Time] = ""
			}
		}
		return pl
	}
*/
func (e Echeance) String() string {
	return fmt.Sprintf("%s %s",
		e.Day.Format(time.DateOnly),
		e.Moment,
	)
}

func (mf *MultiforecastData) FindDaily(id CodeInsee, ech time.Time) *Daily {
	for _, feat := range *mf {
		if feat.Properties.Insee != id {
			continue
		}
		for _, d := range feat.Properties.Dailies {
			if d.Time != ech {
				continue
			}
			return &d
		}
	}
	return nil
}
