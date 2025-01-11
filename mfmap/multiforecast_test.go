package mfmap

import (
	"fmt"
	"gometeo/testutils"
	"regexp"
	"slices"
	"testing"
)

const (
	emptyRegexp    = `^$`
	anyRegexp      = `.*`
	coordsRegexp   = `^([\d\.],?)+$`
	instantsRegexp = `morning,afternoon,evening,night`
)

func TestForecastQuery(t *testing.T) {
	m := testParseHtml(t)

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
	_ = testParseMultiforecast(t)
}

func testParseMultiforecast(t *testing.T) MultiforecastData {

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

func TestPictoNames(t *testing.T) {
	const minLength = 20
	m := MfMap{
		Forecasts: testParseMultiforecast(t),
	}
	pics := m.PictoNames()
	if len(pics) < minLength {
		t.Errorf("picto list is too short (<%d items), %v", minLength, pics)
	}
	if slices.Contains[[]string, string](pics, "") {
		t.Errorf("picto list contains an empty string : %v", pics)
	}
}
