package mfmap

import (
	"fmt"
	"testing"
	"time"

	"gometeo/testutils"
)

func makeMultiforecast(t *testing.T) MultiforecastData {
	j := testutils.MultiforecastReader(t)
	defer j.Close()

	m := MfMap{}
	err := m.ParseMultiforecast(j)
	if err != nil {
		t.Fatal(fmt.Errorf("parseMultiforecast() error: %w", err))
	}
	if len(m.Forecasts) == 0 {
		t.Fatal("parseMultiforecast() returned no data")
	}
	return m.Forecasts
}

func TestByEcheances(t *testing.T) {

	mf := makeMultiforecast(t)
	prevs, err := mf.byEcheance()
	if err != nil {
		t.Fatalf("byEcheance error: %s", err)
	}
	if len(prevs) == 0 {
		t.Error("No forecast found in test data")
	}
}

func inspectGraphdata(t *testing.T, g Graphdata) {
	if g == nil {
		t.Fatal("Graphdata is nil")
	}
	for _, key := range append(forecastsChroniques, dailiesChroniques...) {
		series, ok := g[key]
		if !ok || len(series) == 0 {
			t.Errorf("missing or empty serie: '%s'", key)
			continue
		}
	}
}

func TestToChronique(t *testing.T) {
	mf := makeMultiforecast(t)
	g, err := mf.toChroniques()
	if err != nil {
		t.Fatalf("toChronique() error: %s", err)
	}
	inspectGraphdata(t, g)
}

func TestEcheanceString(t *testing.T) {
	m := morningStr
	d, _ := time.Parse(time.RFC3339, "2024-12-02T15:51:12.000Z")
	e := Echeance{Date: NewDate(d), Moment: MomentName(m)}

	want := fmt.Sprintf("%4d-%02d-%02d %s", d.Year(), d.Month(), d.Day(), m)
	got := e.String()
	if got != want {
		t.Errorf("Echeance.String()='%s' want '%s'", got, want)
	}
}

func TestDateSub(t *testing.T) {
	now := Date{2024, 12, 28}
	tests := []struct {
		y int
		m int
		d int
		want int
	}{
		{ y: 2024, m: 12, d: 20, want: -8},
		{ y: 2024, m: 12, d: 27, want: -1},
		{ y: 2024, m: 12, d: 28, want: 0},
		{ y: 2025, m: 1, d: 1, want: 4},
		{ y: 2025, m: 1, d: 2, want: 5},
		{ y: 2026, m: 1, d: 1, want: 365+4},
	}
	for _, test := range tests {
		d := Date{Year:test.y, Month:time.Month(test.m), Day:test.d}
		got := d.Sub(now)
		if got != test.want {
			t.Errorf("DaysFrom() test : (%s)-(%s) got %d want %d", d, now, got, test.want)
		}
	}
}

func TestFindDaily(t *testing.T) {
	mf := makeMultiforecast(t)

	// these values must be updated after a change in test_data...
	id := codeInsee("751010") // "name": "Paris—1er Arrondissement"
	ech, err := time.Parse(time.RFC3339, "2025-01-02T00:00:00.000Z")
	if err != nil {
		t.Fatal(err)
	}
	d := mf.findDaily(id, Echeance{Date:NewDate(ech)})
	if d == nil {
		t.Fatalf("FindDaily() did not found daily forecast for location '%s' at '%s'", id, ech)
	}
}
