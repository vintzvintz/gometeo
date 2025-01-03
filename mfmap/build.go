package mfmap

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"text/template"
	"time"
)

type JsonMap struct {
	Name     string      `json:"name"`
	Path     string      `json:"path"`
	Idtech   string      `json:"idtech"`
	Taxonomy string      `json:"taxonomy"`
	Bbox     Bbox        `json:"bbox"`
	SubZones geoFeatures `json:"subzones"`
	Prevs    PrevList    `json:"prevs"`
}

type PrevList map[Jour]PrevsAtDay

// relative day from "today" (-1:yesterday, +1 tomorrow, ...)
type Jour int

// data for a day, to be displayed as a row of 4 moments
// use pointers so unavailable entries can be nil
type PrevsAtDay struct {
	Matin     *PrevsAtMoment `json:"matin"`
	AprèsMidi *PrevsAtMoment `json:"après-midi"`
	Soiree    *PrevsAtMoment `json:"soirée"`
	Nuit      *PrevsAtMoment `json:"nuit"`
}

// all available forecasts for a given point in time (moment + day)
type PrevsAtMoment struct {
	Time    time.Time   `json:"echeance"`
	Updated time.Time   `json:"updated"`
	Prevs   []PrevAtPoi `json:"prevs"`
}

// Prevlist key is a composite type
type Echeance struct {
	Moment MomentName
	Day    time.Time // yyyy-mm-dd @ 00-00-00 UTC
}

// forecast data for a single (poi, date) point
type PrevAtPoi struct {
	Title  string      `json:"titre"`
	Coords Coordinates `json:"coords"`
	Prev   *Forecast   `json:"prev"`
	Daily  *Daily      `json:"daily"`
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

// time-series for charts
type Graphdata map[string][]Chronique

// time reference javascript
var jsEpoch = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

var ErrNoSuchData = fmt.Errorf("no such data")

// htmlTemplate is a global html/template for html rendering
// this global variable is set up once at startup by the init() function
var htmlTemplate *template.Template

//go:embed template.html
var templateFile string

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
	htmlTemplate = template.Must(template.New("").Parse(templateFile))
}

func (m *MfMap) buildJson() (*JsonMap, error) {
	j := JsonMap{
		Name:     m.Name(),
		Path:     m.Path(),
		Idtech:   m.Data.Info.IdTechnique,
		Taxonomy: m.Data.Info.Taxonomy,
		SubZones: m.Geography.Features,
		Bbox:     m.Geography.Bbox.Crop(),
		Prevs:    m.Forecasts.byEcheance(),
	}
	return &j, nil
}

func (m *MfMap) BuildJson(wr io.Writer) error {
	obj, err := m.buildJson()
	if err != nil {
		return err
	}
	b, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	_, err = io.Copy(wr, bytes.NewReader(b))
	if err != nil {
		return err
	}
	return nil
}

func (m *MfMap) BuildGraphdata() (Graphdata, error) {
	return m.Forecasts.toChroniques()
}

// momPtr simplifies and reduce duplication of the switch
func (pad *PrevsAtDay) getMomentPtr(e Echeance, prevTime, updateTime time.Time) **PrevsAtMoment {
	var momPtr **PrevsAtMoment
	switch e.Moment {
	case morningStr:
		momPtr = &pad.Matin
	case afternoonStr:
		momPtr = &pad.AprèsMidi
	case eveningStr:
		momPtr = &pad.Soiree
	case nightStr:
		momPtr = &pad.Nuit
	default:
		log.Panicf("invalid moment %s ", e.Moment)
	}

	// create Prev@Moment struct / slice on first pass
	if *momPtr == nil {
		*momPtr = &PrevsAtMoment{
			Time:    prevTime,
			Updated: updateTime,
			Prevs:   []PrevAtPoi{},
		}
	} else {
		// warns if echeances are not unique for different POIs
		// on a same day/moment key
		if (*momPtr).Time != prevTime {
			log.Default().Printf("Inconsistent times for [%s] '%s' != '%s'",
				e, (*momPtr).Time, prevTime)
		}
	}
	return momPtr
}

func (prev *Forecast) getEcheance() Echeance {

	// build an Echeance to use as PrevList key
	year, month, day := prev.Time.Date()
	// "night" moment is equal or after midnight, but displayed with previous day
	if prev.Moment == nightStr {
		day -= 1
	}
	e := Echeance{
		Moment: prev.Moment,
		Day:    time.Date(year, month, day, 0, 0, 0, 0, time.UTC),
	}
	return e
}

func (mf MultiforecastData) byEcheance() PrevList {
	pl := make(PrevList)

	// iterate over POIs, known as "Features" in json data
	for i := range mf {
		prevs := &(mf[i].Properties.Forecasts)
		coords := mf[i].Geometry.Coords
		name := mf[i].Properties.Name
		insee := mf[i].Properties.Insee
		updateTime := mf[i].UpdateTime

		// iterate over echeances
		for j := range *prevs {

			prev := &((*prevs)[j])
			e := prev.getEcheance()
			jour := e.DaysFrom(time.Now())

			// pl[j] is not directly adressable, so we work on a local struct,
			// copy is OK because pad struct contains just 4 pointers.
			// we update pl[j] map entry at loop end.
			pad, ok := pl[jour]
			if !ok {
				pad = PrevsAtDay{}
			}

			momPtr := pad.getMomentPtr(e, prev.Time, updateTime)

			// get daily prev for the day/poi
			daily := mf.findDaily(insee, e.Day)
			if daily == nil {
				log.Default().Printf("Missing daily data for id=%s (%s) echeance %s",
					insee, name, e)
			}

			// wrap forecast and daily together and append to the time-serie of current poi
			pap := PrevAtPoi{
				Title:  name,
				Coords: coords,
				Prev:   prev,
				Daily:  daily,
			}
			(*momPtr).Prevs = append((*momPtr).Prevs, pap)

			// update Prevs@Day in PrevList map
			pl[jour] = pad
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

// MarshalText marshals an Echeance (composite type) to a json object key (string)
func (e Echeance) MarshalText() (text []byte, err error) {
	//return []byte(fmt.Sprintf("%s %s", e.Day, e.Moment)), nil
	return []byte(e.String()), nil
}

// RelDay is the number of days since "now"; may be negative for past Echeances
// used to decide on which "row" of the map is displayed
// now : only year, month and days are considered,
// hours/minutes/seconds and timezone are discarded.
func (e Echeance) DaysFrom(now time.Time) Jour {
	year, month, day := now.Date()
	today := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	d := e.Day.Sub(today).Round(24*time.Hour).Hours() / 24
	return Jour(d)
}

// toChroniques() formats Multiforecastdata into Graphdata
// for client-side charts
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

// TemplateData contains data for htmlTemplate.Execute()
type TemplateData struct {
	HeadDescription string
	HeadTitle       string
	Path            string
	// Breadcrumb      string
	// Idtech          string
}

func (m *MfMap) BuildHtml(wr io.Writer) error {
	return htmlTemplate.Execute(wr, &TemplateData{
		HeadDescription: fmt.Sprintf("Description de %s", m.Data.Info.Name),
		HeadTitle:       fmt.Sprintf("Titre de %s", m.Data.Info.Name),
		Path:            m.Path(),
		//Breadcrumb:      "TODO : generer le breadcrumb",
		//Idtech:          m.Data.Info.IdTechnique,
	})
}

func (mf *MultiforecastData) findDaily(id CodeInsee, day time.Time) *Daily {
	for _, feat := range *mf {
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

type timeStamper interface {
	withTimestamp(data string) (ChroValue, error)
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

// 4 prevs of a day are marshalled into an array (ordered)
// instead of object (unordered) to avoid client-side sorting
func (pad PrevsAtDay) MarshalJSON() ([]byte, error) {
	a := []*PrevsAtMoment{pad.Matin, pad.AprèsMidi, pad.Soiree, pad.Nuit}
	return json.Marshal(a)
}
