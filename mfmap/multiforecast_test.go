package mfmap

import (
	"fmt"
	"regexp"
	"testing"
	"time"
)

const (
	emptyRegexp    = `^$`
	anyRegexp      = `.*`
	coordsRegexp   = `^([\d\.],?)+$`
	instantsRegexp = `morning,afternoon,evening,night`
)

func TestForecastQuery(t *testing.T) {
	m := parseHtml(t, fileHtmlRacine)

	validationRegexps := map[string]string{
		"bbox":       emptyRegexp,
		"begin_time": emptyRegexp,
		"end_time":   emptyRegexp,
		"time":       emptyRegexp,
		"instants":   instantsRegexp,
		"liste_id":   coordsRegexp,
	}

	u, err := m.ForecastURL()
	if err != nil {
		t.Fatalf("forecastURL() error: %s", err)
	}
	values := u.Query()
	for key, expr := range validationRegexps {
		re := regexp.MustCompile(expr)
		got, ok := values[key]
		if !ok {
			t.Fatalf("forecastQuery() does not have key '%s'", key)
		}
		if len(got) != 1 {
			t.Fatalf("forecastQuery()['%s'] has %d values %q, want 1", key, len(got), got)
		}
		if re.Find([]byte(got[0])) == nil {
			t.Errorf("forecastQuery()['%s']='%s' doesnt match '%s'", key, got[0], expr)
		}
	}
}

func TestParseMultiforecast(t *testing.T) {
	_ = testParseMultiforecast(t, fileJsonMultiforecast)
}

func testParseMultiforecast(t *testing.T, name string) MultiforecastData {

	j := openFile(t, name)
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

func TestPictoList(t *testing.T) {
	mf := testParseMultiforecast(t, fileJsonMultiforecast)
	pics := mf.pictoList()
	if len(pics) == 0 {
		t.Errorf("pictoList() returned nothing")
	}
}

func TestByEcheances(t *testing.T) {
/*
	mf := testParseMultiforecast(t, fileJsonMultiforecast)
	prevs := mf.ByEcheance()
	if len(prevs) == 0 {
		t.Errorf("No forecast found in %s", fileJsonMultiforecast)
	}
*/
	/*
	if len(dailies) == 0 {
		t.Errorf("No long-term (daily) forecast found in %s", fileJsonMultiforecast)
	}*/
}


func TestEcheanceString(t *testing.T) {

	m := morningStr
	d, _ := time.Parse( time.RFC3339, "2024-12-02T15:51:12.000Z" )
	e := Echeance{MomentName(m), d}
	want := fmt.Sprintf( "%4d-%02d-%02d %s", d.Year(), d.Month(), d.Day(), m)

	got := e.String()
	if got != want {
		t.Errorf( "Echeance.String()='%s' want '%s'", got, want)
	}
}


func TestFindDaily(t *testing.T) {
	mf := testParseMultiforecast(t, fileJsonMultiforecast)
	id := CodeInsee("440360")   // "name": "Châteaubriant"
	ech, err := time.Parse(time.RFC3339, "2024-12-02T00:00:00.000Z")
	if err != nil {
		t.Fatal(err)
	}
	d := mf.FindDaily( id, ech )
	if d == nil {
		t.Fatalf("FindDaily() did not found daily forecast for location '%s' at '%s'", id, ech )
	}
}