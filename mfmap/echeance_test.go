package mfmap_test

import (
	"gometeo/mfmap"
	"slices"
	"testing"
)

var (
	matin mfmap.MomentName = mfmap.Matin
	aprem mfmap.MomentName = mfmap.Apresmidi
	soir  mfmap.MomentName = mfmap.Soir
	nuit  mfmap.MomentName = mfmap.Nuit

	day1 = mfmap.Date{Year: 2025, Month: 4, Day: 30}
	day2 = mfmap.Date{Year: 2025, Month: 5, Day: 1}
	day3 = mfmap.Date{Year: 2025, Month: 5, Day: 12}
	day4 = mfmap.Date{Year: 2025, Month: 7, Day: 1}
	day5 = mfmap.Date{Year: 2026, Month: 1, Day: 1}

	ech1m = mfmap.Echeance{Date: day1, Moment: matin}
	ech1a = mfmap.Echeance{Date: day1, Moment: aprem}
	ech1n = mfmap.Echeance{Date: day1, Moment: nuit}
	ech2  = mfmap.Echeance{Date: day2, Moment: nuit}
	ech3  = mfmap.Echeance{Date: day3, Moment: soir}
	ech4  = mfmap.Echeance{Date: day4, Moment: aprem}
	ech5  = mfmap.Echeance{Date: day5, Moment: matin}
)

func TestCompareMoments(t *testing.T) {
	tests := []struct {
		a    mfmap.MomentName
		b    mfmap.MomentName
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
		got := mfmap.CompareMoments(test.a, test.b)
		if got != test.want {
			t.Errorf("Echeance.Sub() '%s'- '%s' got %d want %d", test.a, test.b, got, test.want)
		}
	}
}

func TestCompareEcheances(t *testing.T) {
	tests := []struct {
		a    mfmap.Echeance
		b    mfmap.Echeance
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
		got := mfmap.CompareEcheances(test.a, test.b)
		if got != test.want {
			t.Errorf("CompareEcheances('%v', '%v') got %d want %d",
				test.a, test.b, got, test.want)
		}
	}
}

func TestEcheancesSort(t *testing.T) {
	sorted := mfmap.Echeances{ech1m, ech1a, ech1a, ech1n, ech2, ech3, ech4, ech5}

	test := mfmap.Echeances{ech4, ech5, ech1m, ech1a, ech2, ech1n, ech1a, ech3}
	slices.SortFunc[[]mfmap.Echeance](test, mfmap.CompareEcheances)

	for i := range sorted {
		if test[i] != sorted[i] {
			t.Errorf("Sorting echeances failed at index %d got %v want %v", i, test[i], sorted[i])
		}
	}
}
