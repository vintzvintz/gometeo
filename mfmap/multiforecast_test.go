package mfmap

import (
	"fmt"
	"regexp"
	"testing"
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
		"coords":     coordsRegexp,
	}

	u, err := m.forecastURL()
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

	j := openFile(t, "multiforecast.json")
	defer j.Close()

	mf, err := parseMultiforecast(j)
	if err != nil {
		t.Fatal(fmt.Errorf("parseMultiforecast() error: %w", err))
	}
	if len(mf) == 0 {
		t.Fatal("parseMultiforecast() returned no data")
	} 
}
