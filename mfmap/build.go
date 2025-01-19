package mfmap

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"text/template"
	"time"
)

type (
	JsonMap struct {
		Name       string      `json:"name"`
		Path       string      `json:"path"`
		Breadcrumb Breadcrumb  `json:"breadcrumb"`
		Idtech     string      `json:"idtech"`
		Taxonomy   string      `json:"taxonomy"`
		Bbox       Bbox        `json:"bbox"`
		SubZones   geoFeatures `json:"subzones"`
		Prevs      PrevList    `json:"prevs"`
		Chroniques Graphdata   `json:"chroniques"`
	}

	// relative day from "today" (-1:yesterday, +1 tomorrow, ...)
	PrevList map[int]PrevsAtDay

	// data for a day, to be displayed as a row of 4 moments
	// use pointers so unavailable entries can be nil
	PrevsAtDay struct {
		Matin     *PrevsAtMoment `json:"matin"`
		AprèsMidi *PrevsAtMoment `json:"après-midi"`
		Soiree    *PrevsAtMoment `json:"soirée"`
		Nuit      *PrevsAtMoment `json:"nuit"`
	}

	// all available forecasts for a given point in time (moment + day)
	PrevsAtMoment struct {
		Time    time.Time   `json:"echeance"`
		Updated time.Time   `json:"updated"`
		Prevs   []PrevAtPoi `json:"prevs"`
	}

	// forecast data for a single (poi, date) point
	PrevAtPoi struct {
		Title  string      `json:"titre"`
		Coords Coordinates `json:"coords"`
		Prev   *Forecast   `json:"prev"`
		Daily  *Daily      `json:"daily"`
	}

	// Prevlist key is a composite type
	Echeance struct {
		Moment MomentName
		Date   Date // yyyy-mm-dd @ 00-00-00 UTC
	}

	Date struct {
		Year  int
		Month time.Month
		Day   int
	}

	// implemented by Daily and Forecast types
	Echeancer interface {
		Echeance() Echeance
	}
)

type (
	BreadcrumbItem struct {
		Nom  string `json:"nom"`
		Path string `json:"path"`
	}
	Breadcrumb []BreadcrumbItem
)

type (
	// time-series for charts
	Graphdata map[string][]Chronique

	Chronique []ValueTs

	// ValueTs is a float/integer + time stamp pair.
	// FloatTS and IntTs implement custom JSON marshalling suitable for Highchart
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

	IntTs struct {
		ts  time.Time
		val int
	}
)

// time reference javascript
var jsEpoch = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

var ErrNoSuchData = fmt.Errorf("no such data")

//go:embed template.html
var templateFile string

// TemplateData contains data for htmlTemplate.Execute()
type TemplateData struct {
	HeadDescription string
	HeadTitle       string
	Path            string
}

// htmlTemplate is a global html/template for html rendering
// this global variable is set up once at startup by the init() function
var htmlTemplate *template.Template

func init() {
	htmlTemplate = template.Must(template.New("").Parse(templateFile))
}

// series in Forecasts objects

const chroniqueLimitHours = 8 * 24 * time.Hour

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

func (m *MfMap) WriteHtml(wr io.Writer) error {
	return htmlTemplate.Execute(wr, &TemplateData{
		HeadDescription: fmt.Sprintf("Description de %s", m.Data.Info.Name),
		HeadTitle:       fmt.Sprintf("Titre de %s", m.Data.Info.Name),
		Path:            m.Path(),
	})
}

func (m *MfMap) WriteJson(wr io.Writer) error {
	obj, err := m.BuildJson()
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

func (m *MfMap) BuildJson() (*JsonMap, error) {

	prevs, err := m.Forecasts.byEcheance()
	if err != nil {
		return nil, err
	}

	j := JsonMap{
		Name:       m.Name(),
		Path:       m.Path(),
		Breadcrumb: m.Breadcrumb, // not from upstream
		Idtech:     m.Data.Info.IdTechnique,
		Taxonomy:   m.Data.Info.Taxonomy,
		SubZones:   m.Geography.Features, // transfered without modification
		Bbox:       m.Geography.Bbox.Crop(),
		Prevs:      prevs,
		// Chroniques:      // see below
	}
	// highchart disabled for PAYS. Only on DEPTs & REGIONs
	if m.Data.Info.Taxonomy != "PAYS" {
		graphdata, err := m.Forecasts.toChroniques()
		if err != nil {
			return nil, err
		}
		j.Chroniques = graphdata
	}
	//log.Printf("'%s'.Breadcrumb = %v", m.Name(), m.Breadcrumb)
	return &j, nil
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

func (prev Forecast) Echeance() Echeance {
	year, month, day := prev.Time.Date()
	// "night" moment is equal or after midnight, but displayed with previous day
	if prev.Moment == nightStr {
		day -= 1
	}
	return Echeance{
		Moment: prev.Moment,
		Date:   Date{Day: day, Month: month, Year: year},
	}
}

func (d Daily) Echeance() Echeance {
	year, month, day := d.Time.Date()
	return Echeance{
		Moment: "daily",
		Date:   Date{Day: day, Month: month, Year: year},
	}
}

// byEcheance reshapes original data (poi->echeance) into a
// jour->moment->poi structure
// TODO: improve handling of incomplete/invalid mutliforecast ?
func (mf MultiforecastData) byEcheance() (PrevList, error) {
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
			e := prev.Echeance()
			jour := e.Date.DaysFromNow()

			// pl[j] not directly adressable, so we work on a temp copy
			// pl[j] is updated (replaced) at the end of the loop
			pad, ok := pl[jour]
			if !ok {
				pad = PrevsAtDay{}
			}

			momPtr := pad.getMomentPtr(e, prev.Time, updateTime)

			// get daily prev for the day/poi
			daily := mf.findDaily(insee, e.Date)
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
	return pl, nil
}

func (e Echeance) String() string {
	return fmt.Sprintf("%s %s", e.Date, e.Moment)
}

func (d Date) String() string {
	return d.asTime().Format(time.DateOnly)
}

func (d Date) asTime() time.Time {
	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC)
}

func NewDate(t time.Time) Date {
	return Date{Year: t.Year(), Month: t.Month(), Day: t.Day()}
}

// MarshalText marshals an Echeance (composite type) to a json object key (string)
func (e Echeance) MarshalText() (text []byte, err error) {
	//return []byte(fmt.Sprintf("%s %s", e.Day, e.Moment)), nil
	return []byte(e.String()), nil
}

// Sub() returns duration in calendar days from Date ref
// used to decide on which row the map will be displayed
func (d Date) Sub(ref Date) int {
	t1 := d.asTime()
	t2 := ref.asTime()
	diff := t1.Sub(t2).Round(24*time.Hour).Hours() / 24
	return int(math.Round(diff))
}

func (d Date) DaysFromNow() int {
	// TODO : Paramétrer un décalage du changement de date par rapport à 00h00 UTC
	today := NewDate(time.Now())
	return d.Sub(today)
}

// toChroniques() formats Multiforecastdata into Graphdata
// for client-side charts
func (mf MultiforecastData) toChroniques() (Graphdata, error) {
	g := Graphdata{}

	limit := time.Now().Add(chroniqueLimitHours)

	for i := range mf {
		//lieu := mf[i].Properties.Insee

		forecasts := mf[i].Properties.Forecasts
		g1, err := getChroniquesPoi(forecasts, forecastsChroniques)
		if err != nil {
			return nil, err
		}
		for name, chro := range g1 {
			chro = chro.truncateAfter(limit)
			g[name] = append(g[name], chro)
		}

		dailies := mf[i].Properties.Dailies
		g2, err := getChroniquesPoi(dailies, dailiesChroniques)
		if err != nil {
			return nil, err
		}
		for name, chro := range g2 {
			chro = chro.truncateAfter(limit)
			g[name] = append(g[name], chro)
		}
	}
	return g, nil
}

func (c Chronique) truncateAfter(t time.Time) Chronique {
	ret := make(Chronique, 0, len(c))
	for i := range c {
		if c[i].Sub(t) < 0 {
			ret = append(ret, c[i])
		}
	}
	return ret
}

func (mf MultiforecastData) findDaily(id codeInsee, date Date) *Daily {
	day := date.asTime()
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

// getChroniques POI reshapes data for client-side highchart
// * forecasts: list of forecasts (either regular or daily) of a given POI
// * series: names of fields to extract from input forecast data
func getChroniquesPoi[T timeStamper](forecasts []T, series []string) (map[string]Chronique, error) {
	ret := map[string]Chronique{}

seriesLoop:
	// iterate over series names ( T, Tmax, etc... )
	for _, nom := range series {
		var chro = make(Chronique, len(forecasts))
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
			chro[i] = v
		}
		ret[nom] = chro
	}
	return ret, nil
}

func (f Forecast) withTimestamp(data string) (ValueTs, error) {
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
	case Hmin:
		return IntTs{ts, d.Hmin}, nil
	case Hmax:
		return IntTs{ts, d.Hmax}, nil
	case Uv:
		return IntTs{ts, d.Uv}, nil
	default:
		return nil, ErrNoSuchData
	}
}

func timeToJs(t time.Time) int64 {
	return int64(t.Sub(jsEpoch) / time.Millisecond)
}

// 4 prevs of a day are marshalled into an array (ordered)
// instead of object (unordered) to avoid client-side sorting/grouping
func (pad PrevsAtDay) MarshalJSON() ([]byte, error) {
	a := []*PrevsAtMoment{pad.Matin, pad.AprèsMidi, pad.Soiree, pad.Nuit}
	return json.Marshal(a)
}

// MarshalJSON outputs a timestamped float as an array [ts, val]
func (v FloatTs) MarshalJSON() ([]byte, error) {
	s := fmt.Sprintf("[%d, %f]", timeToJs(v.ts), v.val)
	return []byte(s), nil
}

// MarshalJSON outputs a timestamped int as an array [ts, val]
func (v IntTs) MarshalJSON() ([]byte, error) {
	s := fmt.Sprintf("[%d, %d]", timeToJs(v.ts), v.val)
	return []byte(s), nil
}

func (v IntTs) Sub(t time.Time) time.Duration {
	return v.ts.Sub(t)
}

func (v FloatTs) Sub(t time.Time) time.Duration {
	return v.ts.Sub(t)
}
