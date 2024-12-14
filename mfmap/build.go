package mfmap

import (
	"errors"
	"fmt"
	"io"
	"log"
	"text/template"
	"time"
)

type JsonMap struct {
	Name     string
	Idtech   string
	Taxonomy string
	SubZones []geoFeature
	Bbox     Bbox
	Prevs    PrevList
}

type PrevList map[Echeance]PrevsAtEch

// Prevlist key is a composite type
type Echeance struct {
	Moment MomentName
	Day    time.Time // yyyy-mm-dd @ 00-00-00 UTC
}

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
	Short  *Forecast
	Daily  *Daily
}

type ChroValueFloat struct {
	ts  int64 // milliseconds since 1/1/1970
	val float64
}

type ChroValueInt struct {
	ts  int64 // milliseconds since 1/1/1970
	val int
}

// default marshalling is ok
// but need an interface type to implement only once
type ChroValue interface{}

type Chronique []ChroValue

type Graphdata map[string][]Chronique

var jsEpoch = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

var ErrNoSuchData = fmt.Errorf("no such data")

// htmlTemplate is a global html/template for html rendering
// this global variable is set up once at startup by the init() function
var htmlTemplate *template.Template

const templateFile = "template.html"

// series in Forecasts objects
const (
	Temperature   = "T"
	Windspeed     = "Windspeed"
	WindspeedGust = "Windgust"
	Iso0          = "Iso0"
	CloudCover    = "CloudCover"
	Hrel          = "Hrel"
	Psea          = "Psea"
)

const daysChronique = 10

var forecastsChroniques = []string{
	Temperature,
	Windspeed,
	WindspeedGust,
	Iso0,
	CloudCover,
	Hrel,
	Psea,
}

// series in Dailies objects
const (
	Tmin = "Tmin"
	Tmax = "Tmax"
	Hmin = "Hmin"
	Hmax = "Hmax"
	Uv   = "Uv"
)

var dailiesChroniques = []string{
	Tmin,
	Tmax,
	Hmin,
	Hmax,
	Uv,
}

// init() initialises global package-level variables
func init() {
	var err error
	htmlTemplate, err = template.ParseFiles(templateFile)
	if err != nil {
		panic(err)
	}
	_ = htmlTemplate
}

func (m *MfMap) BuildJson() (*JsonMap, error) {
	j := JsonMap{
		Name:     m.Data.Info.Name,
		Idtech:   m.Data.Info.IdTechnique,
		Taxonomy: m.Data.Info.Taxonomy,
		SubZones: m.Geography.Features,
		Bbox:     m.Geography.Bbox.Crop(),
		Prevs:    m.Forecasts.ByEcheance(),
	}
	return &j, nil
}

func (mf MultiforecastData) ByEcheance() PrevList {
	pl := make(PrevList)

	// iterate over POIs, known as "Features" in json data
	for i := range mf {
		shorts := &(mf[i].Properties.Forecasts)
		coords := mf[i].Geometry.Coords
		name := mf[i].Properties.Name
		insee := mf[i].Properties.Insee

		// process short-term forecasts first
		for j := range *shorts {
			short := &((*shorts)[j])

			// build an Echeance to use as PrevList key
			e := Echeance{
				Moment: short.Moment,
				Day: time.Date(
					short.Time.Year(), short.Time.Month(), short.Time.Day(),
					0, 0, 0, 0, time.UTC), // hour, min, sec, nano
			}

			// get previsions@echeance struct
			var pae PrevsAtEch
			_, ok := pl[e]
			if !ok {
				// create Prev@Ech slice if it does not already exist
				pae = PrevsAtEch{
					Time:    short.Time,
					Updated: mf[i].UpdateTime,
					Prevs:   []PrevAtPoi{},
				}
			} else {
				pae = pl[e]
				// warns if echeances are not unique for different POIs
				// on a same day/moment key
				if pae.Time != short.Time {
					log.Default().Printf("Inconsistent times for [%s] '%s' != '%s'",
						e, pae.Time, short.Time)
				}
			}

			// get daily prev for the day/poi
			daily := mf.FindDaily(mf[i].Properties.Insee, e.Day)
			if daily == nil {
				log.Default().Printf("Missing daily data for id=%s (%s) echeance %s",
					insee, name, e)
			}

			// wrap forecast and daily in a struct
			pap := PrevAtPoi{
				Title:  name,
				Coords: coords,
				Short:  short,
				Daily:  daily,
			}
			pae.Prevs = append(pae.Prevs, pap)
			pl[e] = pae
		}
	}
	return pl
}

func (e Echeance) String() string {
	return fmt.Sprintf("%s %s",
		e.Day.Format(time.DateOnly),
		e.Moment,
	)
}

func (m *MfMap) BuildGraphdata() (Graphdata, error) {
	return m.Forecasts.toChroniques()
}

func (m *MfMap) BuildHtml(wr io.Writer) error {
	tmpl, err := template.ParseFiles(templateFile)
	if err != nil {
		return fmt.Errorf("error parsing '%s': %w", templateFile, err)
	}
	err = tmpl.Execute(wr, nil)
	if err != nil {
		return fmt.Errorf("error executing template '%s': %w", templateFile, err)
	}
	return nil
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

type timeStamper interface {
	withTimestamp(data string) (ChroValue, error)
}

func (f Forecast) withTimestamp(data string) (ChroValue, error) {
	ts := int64(f.Time.Sub(jsEpoch) / time.Millisecond)
	switch data {
	case Temperature:
		return ChroValueFloat{ts, f.T}, nil
	case Windspeed:
		return ChroValueInt{ts, f.WindSpeed}, nil
	case WindspeedGust:
		return ChroValueInt{ts, f.WindSpeedGust}, nil
	case CloudCover:
		return ChroValueInt{ts, f.CloudCover}, nil
	case Iso0:
		return ChroValueInt{ts, f.Iso0}, nil
	case Hrel:
		return ChroValueInt{ts, f.Hrel}, nil
	case Psea:
		return ChroValueFloat{ts, f.Pression}, nil
	default:
		return nil, ErrNoSuchData
	}
}

func (d Daily) withTimestamp(data string) (ChroValue, error) {
	ts := int64(d.Time.Sub(jsEpoch) / time.Millisecond)
	switch data {
	case Tmin:
		return ChroValueFloat{ts, d.Tmin}, nil
	case Tmax:
		return ChroValueFloat{ts, d.Tmax}, nil
	case Hmin:
		return ChroValueInt{ts, d.Hmin}, nil
	case Hmax:
		return ChroValueInt{ts, d.Hmax}, nil
	case Uv:
		return ChroValueInt{ts, d.Uv}, nil
	default:
		return nil, ErrNoSuchData
	}
}

func getChroniques[T timeStamper](forecasts []T, series []string) (Graphdata, error) {
	g := Graphdata{}
seriesLoop:
	for _, serie := range series {
		//c, err := getChronique(forecasts, serie)
		var chro = make(Chronique, len(forecasts))
		for i := range forecasts {
			f := forecasts[i]
			v, err := f.withTimestamp(serie)
			if errors.Is(err, ErrNoSuchData) {
				continue seriesLoop // shortcut to next serie
			}
			if err != nil {
				return nil, fmt.Errorf("getChroniques(%s) error: %w", serie, err)
			}
			chro[i] = v
		}
		g[serie] = append(g[serie], chro)
	}
	return g, nil
}

// toChroniques() formats Multiforecastdata into Graphdata
// for client-side plottings
func (mf MultiforecastData) toChroniques() (Graphdata, error) {
	g := Graphdata{}
	for i := range mf {
		//lieu := mf[i].Properties.Insee

		forecasts := mf[i].Properties.Forecasts
		g1, err := getChroniques(forecasts, forecastsChroniques)
		if err != nil {
			return nil, err
		}
		for k, v := range g1 {
			g[k] = v
		}

		dailies := mf[i].Properties.Dailies
		g2, err := getChroniques(dailies, dailiesChroniques)
		if err != nil {
			return nil, err
		}
		for k, v := range g2 {
			g[k] = v
		}
	}
	return g, nil
}
