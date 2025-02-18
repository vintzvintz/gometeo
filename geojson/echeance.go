package geojson

import (
	"encoding/json"
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

	Echeances []Echeance
)

const (
	Matin      = "matin"
	Apresmidi  = "après-midi"
	Soir       = "soirée"
	Nuit       = "nuit"
	Journalier = "daily"
)

// momentsStr is an alias for the 4 moments, not including 'daily'
var momentsStr = []MomentName{Matin, Apresmidi, Soir, Nuit}

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
	if f.Moment == Nuit {
		day -= 1
	}
	return Echeance{
		Moment: f.Moment,
		Date:   Date{Day: day, Month: month, Year: year},
	}
}

// compare two moments
func CompareMoments(a, b MomentName) int {
	var ia, ib int
	for i := range momentsStr {
		if a == momentsStr[i] {
			ia = i
		}
		if b == momentsStr[i] {
			ib = i
		}
	}
	return ia - ib
}

// Compare compares two Echeance structs
// sort a []Echeance in ascending order with slice.SortFunc()
func CompareEcheances(a, b Echeance) int {
	if a.Date.Year != b.Date.Year {
		return a.Date.Year - b.Date.Year
	}
	if a.Date.Month != b.Date.Month {
		return int(a.Date.Month - b.Date.Month)
	}
	if a.Date.Day != b.Date.Day {
		return a.Date.Day - b.Date.Day
	}
	return CompareMoments(a.Moment, b.Moment)
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

func (m *MomentName) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("moment unmarshal error: %w", err)
	}
	allowedNames := []string{Matin, Apresmidi, Soir, Nuit}
	for _, name := range allowedNames {
		if s == name {
			*m = MomentName(s)
			return nil
		}
	}
	return fmt.Errorf("moment '%s' not in known values  %v", s, allowedNames)
}
