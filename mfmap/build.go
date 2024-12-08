package mfmap

import (
	"fmt"
	"log"
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

type Graphdata []struct {
	wesh int
}

func (pl PrevList) toChroniques() Graphdata {
	return nil
}
