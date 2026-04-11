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

func TestTodayDatePivot(t *testing.T) {
	// Pivot is 03:00Z. Row J+0 advances at that instant, every UTC day.
	// DST plays no role (pivot is UTC-anchored).
	tests := []struct {
		name    string
		nowUTC  time.Time
		wantYMD [3]int
	}{
		{"just-before-pivot", time.Date(2026, 1, 15, 2, 59, 59, 0, time.UTC), [3]int{2026, 1, 14}},
		{"at-pivot", time.Date(2026, 1, 15, 3, 0, 0, 0, time.UTC), [3]int{2026, 1, 15}},
		{"just-after-pivot", time.Date(2026, 1, 15, 3, 0, 1, 0, time.UTC), [3]int{2026, 1, 15}},
		{"mid-morning", time.Date(2026, 1, 15, 8, 0, 0, 0, time.UTC), [3]int{2026, 1, 15}},
		{"late-evening", time.Date(2026, 1, 15, 23, 59, 0, 0, time.UTC), [3]int{2026, 1, 15}},
		{"next-midnight", time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC), [3]int{2026, 1, 15}},
		{"next-02h59", time.Date(2026, 1, 16, 2, 59, 0, 0, time.UTC), [3]int{2026, 1, 15}},
		// DST transitions — should behave identically to any other day
		// because the pivot is in UTC.
		{"spring-forward-before-pivot", time.Date(2026, 3, 29, 2, 59, 0, 0, time.UTC), [3]int{2026, 3, 28}},
		{"spring-forward-after-pivot", time.Date(2026, 3, 29, 3, 0, 0, 0, time.UTC), [3]int{2026, 3, 29}},
		{"fall-back-before-pivot", time.Date(2026, 10, 25, 2, 59, 0, 0, time.UTC), [3]int{2026, 10, 24}},
		{"fall-back-after-pivot", time.Date(2026, 10, 25, 3, 0, 0, 0, time.UTC), [3]int{2026, 10, 25}},
		// Month / year rollover across the pivot.
		{"month-rollover", time.Date(2026, 2, 1, 2, 0, 0, 0, time.UTC), [3]int{2026, 1, 31}},
		{"year-rollover", time.Date(2027, 1, 1, 2, 0, 0, 0, time.UTC), [3]int{2026, 12, 31}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			restore := gj.SetNowForTest(func() time.Time { return tc.nowUTC })
			defer restore()
			got := gj.TodayDateForTest()
			want := gj.Date{Year: tc.wantYMD[0], Month: time.Month(tc.wantYMD[1]), Day: tc.wantYMD[2]}
			if got != want {
				t.Errorf("todayDate() at %s got %v want %v", tc.nowUTC, got, want)
			}
		})
	}
}

func TestEcheanceNight(t *testing.T) {
	// 3h du mat le 1r janver
	f := gj.Forecast{
		Moment: gj.Nuit,
		Time:   time.Date(2025, 1, 1, 3, 0, 0, 0, time.UTC),
	}

	want := gj.Date{2024, 12, 31}
	got := f.Echeance().Date

	if want != got {
		t.Errorf("Date d'une échéance nuit le 1r jour du mois got %d-%d-%d, want %d-%d-%d ",
			got.Year, got.Month, got.Day,
			want.Year, want.Month, want.Day)
	}
}
