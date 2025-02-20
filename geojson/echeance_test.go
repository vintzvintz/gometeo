package geojson_test

import (
	"fmt"
	gj "gometeo/geojson"
	"slices"
	"testing"
	"time"
)

var (
	matin gj.MomentName = gj.Matin
	aprem gj.MomentName = gj.Apresmidi
	soir  gj.MomentName = gj.Soir
	nuit  gj.MomentName = gj.Nuit

	day1 = gj.Date{Year: 2025, Month: 4, Day: 30}
	day2 = gj.Date{Year: 2025, Month: 5, Day: 1}
	day3 = gj.Date{Year: 2025, Month: 5, Day: 12}
	day4 = gj.Date{Year: 2025, Month: 7, Day: 1}
	day5 = gj.Date{Year: 2026, Month: 1, Day: 1}

	ech1m = gj.Echeance{Date: day1, Moment: matin}
	ech1a = gj.Echeance{Date: day1, Moment: aprem}
	ech1n = gj.Echeance{Date: day1, Moment: nuit}
	ech2  = gj.Echeance{Date: day2, Moment: nuit}
	ech3  = gj.Echeance{Date: day3, Moment: soir}
	ech4  = gj.Echeance{Date: day4, Moment: aprem}
	ech5  = gj.Echeance{Date: day5, Moment: matin}
)

func TestCompareMoments(t *testing.T) {
	tests := []struct {
		a    gj.MomentName
		b    gj.MomentName
		want int
	}{
		{matin, matin, 0},
		{matin, aprem, -1},
		{matin, soir, -2},
		{matin, nuit, -3},
		{nuit, nuit, 0},
		{nuit, soir, 1},
		{nuit, aprem, 2},
		{nuit, matin, 3},
	}
	for _, test := range tests {
		got := gj.CompareMoments(test.a, test.b)
		if got != test.want {
			t.Errorf("Echeance.Sub() '%s'- '%s' got %d want %d", test.a, test.b, got, test.want)
		}
	}
}

func TestCompareEcheances(t *testing.T) {
	tests := []struct {
		a    gj.Echeance
		b    gj.Echeance
		want int
	}{
		{ech1m, ech1m, 0},
		{ech1a, ech1a, 0},
		{ech1m, ech1a, -1},
		{ech1n, ech1m, 3},
		{ech1m, ech2, -1},
		{ech2, ech3, -11},
		{ech3, ech4, -2},
		{ech4, ech5, -1},
	}
	for _, test := range tests {
		got := gj.CompareEcheances(test.a, test.b)
		if got != test.want {
			t.Errorf("CompareEcheances('%v', '%v') got %d want %d",
				test.a, test.b, got, test.want)
		}
	}
}

func TestEcheancesSort(t *testing.T) {
	sorted := gj.Echeances{ech1m, ech1a, ech1a, ech1n, ech2, ech3, ech4, ech5}

	test := gj.Echeances{ech4, ech5, ech1m, ech1a, ech2, ech1n, ech1a, ech3}
	slices.SortFunc[[]gj.Echeance](test, gj.CompareEcheances)

	for i := range sorted {
		if test[i] != sorted[i] {
			t.Errorf("Sorting echeances failed at index %d got %v want %v", i, test[i], sorted[i])
		}
	}
}

func TestEcheanceString(t *testing.T) {
	m := gj.Matin
	d, _ := time.Parse(time.RFC3339, "2024-12-02T15:51:12.000Z")
	e := gj.Echeance{Date: gj.NewDate(d), Moment: gj.MomentName(m)}

	want := fmt.Sprintf("%4d-%02d-%02d %s", d.Year(), d.Month(), d.Day(), m)
	got := e.String()
	if got != want {
		t.Errorf("Echeance.String()='%s' want '%s'", got, want)
	}
}

func TestDateSub(t *testing.T) {
	now := gj.Date{2024, 12, 28}
	tests := []struct {
		y    int
		m    int
		d    int
		want int
	}{
		{y: 2024, m: 12, d: 20, want: -8},
		{y: 2024, m: 12, d: 27, want: -1},
		{y: 2024, m: 12, d: 28, want: 0},
		{y: 2025, m: 1, d: 1, want: 4},
		{y: 2025, m: 1, d: 2, want: 5},
		{y: 2026, m: 1, d: 1, want: 365 + 4},
	}
	for _, test := range tests {
		d := gj.Date{Year: test.y, Month: time.Month(test.m), Day: test.d}
		got := d.Sub(now)
		if got != test.want {
			t.Errorf("DaysFrom() test : (%s)-(%s) got %d want %d", d, now, got, test.want)
		}
	}
}
