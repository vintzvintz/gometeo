package geojson

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"slices"
	"time"
)

type (
	// time-series for charts
	Graphdata map[NomSerie]Chroniques

	NomSerie string

	Chroniques map[codeInsee]Chronique

	Chronique []ValueTs

	// ValueTs is a (float/integer + timestamp) pair.
	// concrete types FloatTs, IntTs, FloatRangeTs, IntRangeTs
	// have custom JSON marshalling methods suitable for Highchart
	ValueTs interface {
		json.Marshaler
		Sub(time.Time) time.Duration
		Ts() time.Time
	}

	// timeStamper yields TsValues from a specified 'series' name'
	timeStamper interface {
		withTimestamp(series NomSerie) (ValueTs, error)
	}

	FloatTs struct {
		ts  time.Time
		val float64
	}

	FloatRangeTs struct {
		ts          time.Time
		min         float64
		max         float64
		offsetHours int
	}

	IntTs struct {
		ts  time.Time
		val int
	}

	IntRangeTs struct {
		ts          time.Time
		min         int
		max         int
		offsetHours int
	}
)

const (
	// series in Forecast objects
	Temperature   = "T"
	Ressenti      = "Ress"
	WindSpeed     = "WindSpeed"
	WindSpeedGust = "WindSpeedGust"
	Iso0          = "Iso0"
	CloudCover    = "Cloud"
	Hrel          = "Hrel"
	Psea          = "Psea"

	// series in Dailies objects
	Uv     = "Uv"
	Trange = "Trange"
	Hrange = "Hrange"

	// shift min/max points position from 00h00
	dailyOffset = 8 // hours
)

var forecastsChroniques = []NomSerie{
	Temperature,
	Ressenti,
	WindSpeed,
	WindSpeedGust,
	Iso0,
	CloudCover,
	Hrel,
	Psea,
}

var dailiesChroniques = []NomSerie{
	Trange,
	Hrange,
	Uv,
}

// javascript epoch
var jsEpoch = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

var ErrNoSuchData = fmt.Errorf("no such data")

const chroniqueMaxDays = 11

// toChroniques() formats Multiforecastdata into Graphdata
// for client-side charts
func (mf MultiforecastData) BuildChroniques() (Graphdata, error) {
	g := Graphdata{}

	for i := range mf {
		codeInsee := mf[i].Properties.Insee

		forecasts := mf[i].Properties.Forecasts
		g1, err := getChroniquesPoi(forecasts, forecastsChroniques)
		if err != nil {
			return nil, err
		}
		for name, c := range g1 {
			chros := g[name]
			if chros == nil {
				chros = make(Chroniques)
			}
			chros[codeInsee] = c
			g[name] = chros
		}

		dailies := mf[i].Properties.Dailies
		g2, err := getChroniquesPoi(dailies, dailiesChroniques)
		if err != nil {
			return nil, err
		}
		for name, c := range g2 {
			chros := g[name]
			if chros == nil {
				chros = make(Chroniques)
			}
			chros[codeInsee] = c
			g[name] = chros
		}
	}
	return g, nil
}

func (c Chroniques)MarshalJSON()([]byte, error) {
	tmp := []Chronique{}
	for insee := range c {
		tmp = append(tmp, c[insee])
	}
	return json.Marshal(tmp)
}

func (g Graphdata) Merge(old Graphdata, dayMin, dayMax int) {

	for nom := range g {
		// skip series not present in old
		oldSerie, ok := old[nom]
		if !ok {
			continue
		}
		serie := g[nom]
		for insee := range serie {
			oldChro, ok := oldSerie[insee]
			// skip places (codeInsee) not present in old
			if !ok {
				continue
			}
			newChro := serie[insee]
			serie[insee] = mergeChronique(newChro, oldChro, dayMin, dayMax)
		}
		g[nom] = serie
	}
}

func mergeChronique(new, old Chronique, dayMin, dayMax int) Chronique {

	// use a temp map keyed by unix timestamp to merge old into new
	merged := make(map[int64]ValueTs)

	// fill with old data, filtered by dayMin, dayMax
	for _, v := range old {
		age := time.Since(v.Ts()) / time.Hour

		if int(age) < 24*dayMin || int(age) > 24*dayMax {
			continue
		}
		merged[v.Ts().Unix()] = v
	}

	// insert all new data overwriting old
	for _, v := range new {
		//ts := v.Ts().Unix()
		merged[v.Ts().Unix()] = v
	}

	// sort timestamps
	timestamps := make([]int64, 0, len(merged))
	for ts := range merged {
		timestamps = append(timestamps, ts)
	}
	slices.Sort[[]int64](timestamps)

	ret := make(Chronique, 0, len(merged))
	for _, ts := range timestamps {
		ret = append(ret, merged[ts])
	}
	return ret
}

// getChroniques POI reshapes data for client-side highchart
// * forecasts: list of forecasts (either regular or daily) of a given POI
// * series: names of fields to extract from input forecast data
func getChroniquesPoi[T timeStamper](forecasts []T, series []NomSerie) (map[NomSerie]Chronique, error) {
	ret := map[NomSerie]Chronique{}
seriesLoop:
	// iterate over series names ( T, Tmax, etc... )
	for _, nom := range series {
		var chro = make(Chronique, 0, len(forecasts))
		//iterate over forecasts of the POI at all available echeances
		for i := range forecasts {
			f := forecasts[i]
			v, err := f.withTimestamp(nom)
			if errors.Is(err, ErrNoSuchData) {
				ret[nom] = nil
				log.Printf("series not found: '%s'", nom)
				continue seriesLoop // shortcut to next serie
			}
			if err != nil {
				return nil, fmt.Errorf("getChroniques(%s) error: %w", nom, err)
			}
			// ignore missing values
			if v == nil {
				continue
			}
			// ignore data after configured limit
			if v.Sub(time.Now()) > chroniqueMaxDays*24*time.Hour {
				continue
			}
			chro = append(chro, v)
		}
		ret[nom] = chro
	}
	return ret, nil
}

func (f Forecast) withTimestamp(data NomSerie) (ValueTs, error) {
	if f.LongTerme {
		return nil, nil
	}
	ts := f.Time
	switch data {
	case Temperature:
		return FloatTs{ts, f.T}, nil
	case Ressenti:
		return FloatTs{ts, f.TWindchill}, nil
	case WindSpeed:
		return IntTs{ts, f.WindSpeed}, nil
	case WindSpeedGust:
		return IntTs{ts, f.WindSpeedGust}, nil
	case CloudCover:
		return IntTs{ts, f.CloudCover}, nil
	case Iso0:
		return IntTs{ts, f.Iso0}, nil
	case Hrel:
		return IntTs{ts, f.Hrel}, nil
	case Psea:
		return FloatTs{ts, f.Pression}, nil
	default:
		return nil, ErrNoSuchData
	}
}

func (d Daily) withTimestamp(data NomSerie) (ValueTs, error) {
	ts := d.Time
	switch data {
	case Trange:
		return FloatRangeTs{ts, d.Tmin, d.Tmax, dailyOffset}, nil
	case Hrange:
		return IntRangeTs{ts, d.Hmin, d.Hmax, dailyOffset}, nil
	case Uv:
		return IntTs{ts, d.Uv}, nil
	default:
		return nil, ErrNoSuchData
	}
}

func timeToJs(t time.Time) int64 {
	return int64(t.Sub(jsEpoch) / time.Millisecond)
}

// MarshalJSON outputs a timestamped float as an array [ts, val]
func (v FloatTs) MarshalJSON() ([]byte, error) {
	s := fmt.Sprintf("[%d, %f]", timeToJs(v.ts), v.val)
	return []byte(s), nil
}

// MarshalJSON outputs a timestamped float as an array [ts, min, max]
func (v FloatRangeTs) MarshalJSON() ([]byte, error) {
	t := timeToJs(v.ts.Add(time.Duration(v.offsetHours) * time.Hour))
	s := fmt.Sprintf("[%d, %f, %f]", t, v.min, v.max)
	return []byte(s), nil
}

// MarshalJSON outputs a timestamped int as an array [ts, val]
func (v IntTs) MarshalJSON() ([]byte, error) {
	s := fmt.Sprintf("[%d, %d]", timeToJs(v.ts), v.val)
	return []byte(s), nil
}

// MarshalJSON outputs a timestamped float as an array [ts, min, max]
func (v IntRangeTs) MarshalJSON() ([]byte, error) {
	t := timeToJs(v.ts.Add(time.Duration(v.offsetHours) * time.Hour))
	s := fmt.Sprintf("[%d, %d, %d]", t, v.min, v.max)
	return []byte(s), nil
}

func (v IntTs) Sub(t time.Time) time.Duration        { return v.ts.Sub(t) }
func (v FloatTs) Sub(t time.Time) time.Duration      { return v.ts.Sub(t) }
func (v IntRangeTs) Sub(t time.Time) time.Duration   { return v.ts.Sub(t) }
func (v FloatRangeTs) Sub(t time.Time) time.Duration { return v.ts.Sub(t) }

func (v IntTs) Ts() time.Time        { return v.ts }
func (v FloatTs) Ts() time.Time      { return v.ts }
func (v IntRangeTs) Ts() time.Time   { return v.ts }
func (v FloatRangeTs) Ts() time.Time { return v.ts }
