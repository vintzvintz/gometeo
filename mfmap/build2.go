package mfmap

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"
)

type (
	// time-series for charts
	Graphdata map[string][]Chronique

	Chronique []ValueTs

	// ValueTs is a (float/integer + timestamp) pair.
	// concrete types FloatTs, IntTs, FloatRangeTs, IntRangeTs
	// have custom JSON marshalling methods suitable for Highchart
	ValueTs interface {
		json.Marshaler
		Sub(time.Time) time.Duration
	}

	// timeStamper yields TsValues from a specified 'series' name'
	timeStamper interface {
		withTimestamp(series string) (ValueTs, error)
	}

	FloatTs struct {
		ts  time.Time
		val float64
	}

	FloatRangeTs struct {
		ts  time.Time
		min float64
		max float64
		offsetHours int 
	}

	IntTs struct {
		ts  time.Time
		val int
	}

	IntRangeTs struct {
		ts  time.Time
		min int
		max int
		offsetHours int 
	}
)

const (
	Temperature   = "T"
	Ressenti      = "Ress"
	WindSpeed     = "WindSpeed"
	WindSpeedGust = "WindSpeedGust"
	Iso0          = "Iso0"
	CloudCover    = "Cloud"
	Hrel          = "Hrel"
	Psea          = "Psea"
)

var forecastsChroniques = []string{
	Temperature,
	Ressenti,
	WindSpeed,
	WindSpeedGust,
	Iso0,
	CloudCover,
	Hrel,
	Psea,
}

// series in Dailies objects
const (
	Tmin   = "Tmin"
	Tmax   = "Tmax"
	Hmin   = "Hmin"
	Hmax   = "Hmax"
	Uv     = "Uv"
	Trange = "Trange"
	Hrange = "Hrange"
)

var dailiesChroniques = []string{
	Trange,
//	Tmin,
//	Tmax,
	Hrange,
//	Hmin,
//	Hmax,
	Uv,
}

// javascript epoch
var jsEpoch = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

var ErrNoSuchData = fmt.Errorf("no such data")

const chroniqueMaxDays = 11

// toChroniques() formats Multiforecastdata into Graphdata
// for client-side charts
func (mf MultiforecastData) toChroniques() (Graphdata, error) {
	g := Graphdata{}

	for i := range mf {
		//lieu := mf[i].Properties.Insee

		forecasts := mf[i].Properties.Forecasts
		g1, err := getChroniquesPoi(forecasts, forecastsChroniques)
		if err != nil {
			return nil, err
		}
		for name, chro := range g1 {
			g[name] = append(g[name], chro)
		}

		dailies := mf[i].Properties.Dailies
		g2, err := getChroniquesPoi(dailies, dailiesChroniques)
		if err != nil {
			return nil, err
		}
		for name, chro := range g2 {
			g[name] = append(g[name], chro)
		}
	}
	return g, nil
}

// getChroniques POI reshapes data for client-side highchart
// * forecasts: list of forecasts (either regular or daily) of a given POI
// * series: names of fields to extract from input forecast data
func getChroniquesPoi[T timeStamper](forecasts []T, series []string) (map[string]Chronique, error) {
	ret := map[string]Chronique{}
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
			if v.Sub( time.Now() ) > chroniqueMaxDays * 24 * time.Hour {
				continue
			}
			chro = append(chro, v)
		}
		ret[nom] = chro
	}
	return ret, nil
}

func (f Forecast) withTimestamp(data string) (ValueTs, error) {
	if f.LongTerme {
		return nil,nil
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

func (d Daily) withTimestamp(data string) (ValueTs, error) {
	ts := d.Time
	switch data {
	case Tmin:
		return FloatTs{ts, d.Tmin}, nil
	case Tmax:
		return FloatTs{ts, d.Tmax}, nil
	case Trange:
		return FloatRangeTs{ts, d.Tmin, d.Tmax, 8}, nil
	case Hmin:
		return IntTs{ts, d.Hmin}, nil
	case Hmax:
		return IntTs{ts, d.Hmax}, nil
	case Hrange:
		return IntRangeTs{ts, d.Hmin, d.Hmax, 8}, nil
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
	t :=  timeToJs(v.ts.Add( time.Duration(v.offsetHours) * time.Hour))
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
	t :=  timeToJs(v.ts.Add( time.Duration(v.offsetHours) * time.Hour))
	s := fmt.Sprintf("[%d, %d, %d]", t, v.min, v.max)
	return []byte(s), nil
}

func (v IntTs) Sub(t time.Time) time.Duration {
	return v.ts.Sub(t)
}

func (v FloatTs) Sub(t time.Time) time.Duration {
	return v.ts.Sub(t)
}

func (v IntRangeTs) Sub(t time.Time) time.Duration {
	return v.ts.Sub(t)
}

func (v FloatRangeTs) Sub(t time.Time) time.Duration {
	return v.ts.Sub(t)
}
