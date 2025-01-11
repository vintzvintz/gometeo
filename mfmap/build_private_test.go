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
	e := Echeance{MomentName(m), d}
	want := fmt.Sprintf("%4d-%02d-%02d %s", d.Year(), d.Month(), d.Day(), m)

	got := e.String()
	if got != want {
		t.Errorf("Echeance.String()='%s' want '%s'", got, want)
	}
}

func TestDaysFrom(t *testing.T) {
	now := time.Date(2024, 12, 28, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		e    Echeance
		want Jour
	}{
		{e: Echeance{Day: time.Date(2024, 12, 26, 0, 0, 0, 0, time.UTC)}, want: -2},
		{e: Echeance{Day: time.Date(2024, 12, 27, 0, 0, 0, 0, time.UTC)}, want: -1},
		{e: Echeance{Day: time.Date(2024, 12, 28, 0, 0, 0, 0, time.UTC)}, want: 0},
		{e: Echeance{Day: time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)}, want: 3},
		{e: Echeance{Day: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}, want: 4},
		{e: Echeance{Day: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)}, want: 5},
		{e: Echeance{Day: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}, want: 365 + 4},
	}
	for i, test := range tests {
		got := test.e.DaysFrom(now)
		if got != test.want {
			t.Errorf("DaysFrom() test #%d : %s -> %s got %d want %d", i, now, test.e, got, test.want)
		}
	}
}

func TestFindDaily(t *testing.T) {
	mf := makeMultiforecast(t)

	// these values must be updated after a change in test_data...
	id := CodeInsee("751010") // "name": "Paris—1er Arrondissement"
	ech, err := time.Parse(time.RFC3339, "2025-01-02T00:00:00.000Z")
	if err != nil {
		t.Fatal(err)
	}
	d := mf.findDaily(id, ech)
	if d == nil {
		t.Fatalf("FindDaily() did not found daily forecast for location '%s' at '%s'", id, ech)
	}
}
