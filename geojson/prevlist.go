package geojson

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

type (
	// relative day from "today" (-1:yesterday, +1 tomorrow, ...)
	PrevList map[Date]prevsAtDay

	// data for a day, to be displayed as a row of 4 moments or just a daily map
	prevsAtDay map[MomentName]prevsAtMoment

	// all available forecasts for a given point in time (moment + day)
	prevsAtMoment struct {
		Time    time.Time   `json:"echeance"`
		Updated time.Time   `json:"updated"`
		Prevs   prevsAtPois `json:"prevs"`
	}

	prevsAtPois map[codeInsee]prevAtPoi

	// forecast data for a single (poi, moment) point
	prevAtPoi struct {
		Title  string        `json:"titre"`
		Coords Coordinates   `json:"coords"`
		Prev   forecastBuild `json:"prev"` //  *Forecast + *Daily
	}

	// intermediate struct for data reshaping
	forecastBuild struct {
		f *Forecast
		d *Daily
	}
)

type prevTerme int

const (
	termeUnknown prevTerme = iota
	termeTendance
	termeCourt
)

// marshal a prevList (indexed by calendar date) into
// map indexed by number of relative days
func (pl PrevList) MarshalJSON() ([]byte, error) {
	var data = make(map[int]prevsAtDay)
	for d := range pl {
		data[d.DaysFromNow()] = pl[d]
	}
	return json.Marshal(data)
}

type featInfo struct {
	coords     Coordinates
	name       string
	insee      codeInsee
	updateTime time.Time
}

func (pl PrevList) Merge(old PrevList, dayMin, dayMax int) {
	// iterate over old prevs to fill missing slots in pl
	for date := range old {
		// ignore dates outside of requested time window
		n := date.DaysFromNow()
		if n < dayMin || n > dayMax {
			continue
		}
		padOld := old[date]
		padNew := pl[date]
		// create missing prevs slot in new
		// required if pastDays < 0 because upstream does not send history
		if padNew == nil {
			padNew = make(prevsAtDay)
		}
		// copy old[date][moment] in padNew, only if moment is missing in new
		for moment := range padOld {
			if _, ok := padNew[moment]; ok {
				continue
			}
			padNew[moment] = padOld[moment]
		}
		// store merged padNew back into pl
		pl[date] = padNew
	}
}

// byEcheance reshapes original data (poi->echeance) into a
// reversed jour->moment->poi structure
// TODO: improve handling of incomplete/invalid mutliforecast
func (mf MultiforecastData) BuildPrevs() (PrevList, error) {
	pl := make(PrevList)

	// iterate over POIs, known as "Features" in json data
	for i := range mf {
		forecasts := mf[i].Properties.Forecasts
		fi := featInfo{
			coords:     mf[i].Geometry.Coords,
			name:       mf[i].Properties.Name,
			insee:      mf[i].Properties.Insee,
			updateTime: mf[i].UpdateTime,
		}

		// iterate over echeances
		for j := range forecasts {
			f := &(forecasts[j])
			e := f.Echeance()

			// create PrevAtDay struct on first pass
			pad, ok := pl[e.Date]
			if !ok {
				pad = make(map[MomentName]prevsAtMoment)
				// pad is a map , not a local copy
				// mutations of pad are mirrored in pl[jour]
				pl[e.Date] = pad
			}

			// accumulate Daily prev into PrevAtDay
			d := mf.findDaily(fi.insee, e)
			if d == nil {
				//log.Printf("Missing daily data for id=%s (%s) echeance %s", fi.insee, fi.name, e)
				continue
			}
			pad.processPrev(Journalier, fi, forecastBuild{nil, d})

			// accumulate Forecast into PrevAtDay
			pad.processPrev(e.Moment, fi, forecastBuild{f, d})
		}
	}
	return pl, nil
}

func (mf MultiforecastData) findDaily(id codeInsee, e Echeance) *Daily {
	day := e.Date.asTime()
	for _, feat := range mf {
		if feat.Properties.Insee != id {
			continue
		}
		for _, d := range feat.Properties.Dailies {
			if d.Time != day {
				continue
			}
			return &d
		}
	}
	return nil
}

func (pad prevsAtDay) processPrev(m MomentName, fi featInfo, fb forecastBuild) {

	// create daily PrevAtMoment struct on first pass
	pam, ok := pad[m]
	if !ok {
		pam = prevsAtMoment{
			// most (all ?) maps have less than 50 geofeatures
			Prevs: make(prevsAtPois, 50),
		}
	}
	// TODO warn if d.Time is not unique among other pam.Prevs
	if fb.f != nil {
		pam.Time = fb.f.Time
	} else if fb.d != nil {
		pam.Time = fb.d.Time
	}
	pam.Updated = fi.updateTime
	pam.Prevs[fi.insee] = prevAtPoi{
		Title:  fi.name,
		Coords: fi.coords,
		Prev:   fb,
	}

	// pam is a local value of a PrevsAtMoment struct
	// write back into PrevsAtDay map to avoid losing updates
	pad[m] = pam
}

func (fb forecastBuild) MarshalJSON() ([]byte, error) {

	type marshallPrev struct {
		// from Forecast
		// TODO : omitempty and pointer types
		Moment MomentName `json:"moment_day"`
		Time   time.Time  `json:"time"`
		T      float64    `json:"T"`

		TWindchill    float64 `json:"T_windchill"`
		WindSpeed     int     `json:"wind_speed"`
		WindSpeedGust int     `json:"wind_speed_gust"`
		WindDirection int     `json:"wind_direction"`
		WindIcon      string  `json:"wind_icon"`
		//Iso0      int     `json:"iso0"`
		CloudCover  int     `json:"total_cloud_cover"`
		WeatherIcon string  `json:"weather_icon"`
		WeatherDesc string  `json:"weather_description"`
		Hrel        int     `json:"relative_humidity"`
		Pression    float64 `json:"P_sea"`
		Confiance   int     `json:"weather_confidence_index"`

		// from Daily
		//Time   time.Time `json:"time"`
		Tmin float64 `json:"T_min"`
		Tmax float64 `json:"T_max"`
		Hmin int     `json:"relative_humidity_min"`
		Hmax int     `json:"relative_humidity_max"`
		Uv   int     `json:"uv_index"`
		//WeatherIcon string    `json:"daily_weather_icon"`
		//WeatherDesc string    `json:"daily_weather_description"`

		LongTerme bool `json:"long_terme"`
	}

	// alias
	f, d := fb.f, fb.d

	// DEBUG : catch a production bug
	if d == nil && f == nil {
		return nil, fmt.Errorf("forecastBuild has 2 nil pointers")
	}
	if d == nil {
		return nil, fmt.Errorf("missing daily prev with forecast %s,", fb.f.describe())
	}

	// basic init with fields for the Daily version (long-term)
	obj := marshallPrev{
		Time:        d.Time,
		Tmin:        d.Tmin,
		Tmax:        d.Tmax,
		Hmin:        d.Hmin,
		Hmax:        d.Hmax,
		Uv:          d.Uv,
		WeatherIcon: d.WeatherIcon,
		WeatherDesc: d.WeatherDesc,
		LongTerme:   true,
	}
	// updates for the reguar version
	if f != nil && !f.LongTerme {
		obj.LongTerme = false
		obj.Moment = f.Moment
		obj.Time = f.Time /// overwrites long term
		obj.T = f.T
		obj.TWindchill = f.TWindchill
		obj.WindSpeed = f.WindSpeed
		obj.WindSpeedGust = f.WindSpeedGust
		obj.WindDirection = f.WindDirection
		obj.WindIcon = f.WindIcon
		obj.CloudCover = f.CloudCover
		obj.WeatherIcon = f.WeatherIcon // overwrites long term
		obj.WeatherDesc = f.WeatherDesc // overwrites long term
		obj.Hrel = f.Hrel
		obj.Pression = f.Pression
		obj.Confiance = f.Confiance
	}
	return json.Marshal(obj)
}

func (f *Forecast) describe() string {
	return fmt.Sprintf("time %v", f.Time)
}

// marshal (unordered) PrevAtDay maps into an (ordered) json array
// avoid putting code about moments names and ordering into front-end
// either 1 single daily map, or 4 matin/am/soir/nuit maps
// missing maps (for example missing history) are filled by a JSON null value
func (pad prevsAtDay) MarshalJSON() ([]byte, error) {

	// local type for customizing PrevAtDay marshalling
	type marshallRow struct {
		LongTerme bool             `json:"long_terme"`
		Maps      []*prevsAtMoment `json:"maps"` // pointer is nil for missing moments
	}
	row := marshallRow{
		// LongTerme: false
		Maps: make([]*prevsAtMoment, 0, 4),
	}

	//insert maps in a fixed order
	var tendance bool
	for _, m := range momentsStr {
		pam, ok := pad[m]
		if !ok {
			row.Maps = append(row.Maps, nil)
			continue
		}
		row.Maps = append(row.Maps, &pam)
		// skip remaining moments if at least one map is long terme
		if ok && (pam.terme() == termeTendance) {
			tendance = true
			break
		}
	}
	// just send a single daily map in 'tendance" mode
	if tendance {
		pam := pad[Journalier]
		row.LongTerme = true
		row.Maps = []*prevsAtMoment{&pam}
	}
	return json.Marshal(row)
}

// determines if a moment is normal, long-term, or unknown
func (pam prevsAtMoment) terme() prevTerme {
	var poiLong, poiCourt int
	for _, p := range pam.Prevs {
		if p.Prev.f == nil {
			continue
		}
		if p.Prev.f.LongTerme {
			poiLong++
		} else {
			poiCourt++
		}
	}
	// normal situations
	if (poiCourt == 0) && (poiLong > 0) {
		return termeTendance
	}
	if (poiCourt > 0) && (poiLong == 0) {
		return termeCourt
	}

	// unexpected situations
	if (poiCourt > 0) && (poiLong > 0) {
		log.Printf("mélange court/long terme entre différents POI au moment %v", pam.Time)
	}
	if (poiCourt == 0) && (poiLong == 0) {
		log.Printf("aucune donnée disponible au moment %v", pam.Time)
	}
	return termeUnknown
}

// marshal map into a json array, codeInsee is not used by frontend
func (prevs prevsAtPois) MarshalJSON() ([]byte, error) {
	tmp := make([]prevAtPoi, 0, len(prevs))
	for _, p := range prevs {
		tmp = append(tmp, p)
	}
	return json.Marshal(tmp)
}
