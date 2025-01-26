package mfmap

import (
	"fmt"
	"math"
	"time"
)

type (
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

// determines Echeance of a Daily
func (d Daily) Echeance() Echeance {
	year, month, day := d.Time.Date()
	return Echeance{
		Moment: "daily",
		Date:   Date{Day: day, Month: month, Year: year},
	}
}

// determines Echeance of a Forecast
// "night" is after midnight, but displayed with previous day
func (f Forecast) Echeance() Echeance {
	year, month, day := f.Time.Date()
	if f.Moment == nightStr {
		day -= 1
	}
	return Echeance{
		Moment: f.Moment,
		Date:   Date{Day: day, Month: month, Year: year},
	}
}

func (e Echeance) String() string {
	return fmt.Sprintf("%s %s", e.Date, e.Moment)
}

func (d Date) String() string {
	return d.asTime().Format(time.DateOnly)
}

// MarshalText marshals an Echeance (composite type) to a json object key (string)
func (e Echeance) MarshalText() (text []byte, err error) {
	//return []byte(fmt.Sprintf("%s %s", e.Day, e.Moment)), nil
	return []byte(e.String()), nil
}

func (d Date) asTime() time.Time {
	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC)
}

func NewDate(t time.Time) Date {
	return Date{Year: t.Year(), Month: t.Month(), Day: t.Day()}
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
