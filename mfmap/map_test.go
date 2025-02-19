package mfmap_test

import (
	"bytes"
	"io"
	"regexp"
	"strings"
	"testing"

	"gometeo/appconf"
	"gometeo/mfmap"
	"gometeo/testutils"
)

func TestParseHtml(t *testing.T) {
	const want = "/meteo-france"
	m := testParseHtml(t)
	// check some content
	if m.Data.Info.Path != "/meteo-france" {
		t.Errorf("MfMap.ParseHtml() Info.Path='%s' want '%s'", m.Data.Info.Path, want)
	}
}

func TestMapParseFail(t *testing.T) {
	tests := map[string]string{
		"missingJsonTag": `
<html>
<head><title>JsonReader test</title></head>
<body>
 <script>wesh</script>
<body>
</html>`,
		"emptyJson": `
<html>
<head><title>JsonReader test</title></head>
<body>
 <script type="application/json" data-drupal-selector="drupal-settings-json"></script>
<body>
</html>`,
		"invalidJson": `
<html>
<head><title>JsonReader test</title></head>
<body>
 <script type="application/json" data-drupal-selector="drupal-settings-json">{invalid json syntax}</script>
<body>
</html>`,
	}
	m := mfmap.MfMap{}
	for name, html := range tests {
		t.Run(name, func(t *testing.T) {
			r := strings.NewReader(html)
			err := m.ParseHtml(r)
			if err == nil {
				t.Error("MfMap.Parse(): error expected")
			}
		})
	}
}

func testParseHtml(t *testing.T) *mfmap.MfMap {
	f := testutils.HtmlReader(t)
	defer f.Close()

	m := mfmap.MfMap{}
	if err := m.ParseHtml(f); err != nil {
		t.Fatalf("MfMap.ParseHtml() error: %s", err)
	}
	return &m
}

const (
	apiBaseURL       = "https://rpcache-aa.meteofrance.com/internet2018client/2.0"
	apiMultiforecast = "/multiforecast"
)

func TestApiUrl(t *testing.T) {

	tests := map[string]struct {
		path string
		want string
	}{
		"racine": {
			path: "",
			want: apiBaseURL,
		},
		"slash": {
			path: "/",
			want: apiBaseURL + "/",
		},
		"multiforecast": {
			path: apiMultiforecast,
			want: apiBaseURL + apiMultiforecast,
		},
	}

	m := testParseHtml(t)

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			u, err := m.ApiUrl(test.path, nil)
			if err != nil {
				t.Fatalf("ApiURL() error : %s", err)
			}
			got := u.String()
			if got != test.want {
				t.Errorf("ApiURL()='%s' want '%s'", got, test.want)
			}
		})
	}
}

// buildTestMap returns a JsonMap structure filled form test files

func testBuildMap(t *testing.T) *mfmap.MfMap {
	var m mfmap.MfMap
	if err := m.ParseHtml(testutils.HtmlReader(t)); err != nil {
		t.Error(err)
	}
	if err := m.ParseGeography(testutils.GeoCollectionReader(t)); err != nil {
		t.Error(err)
	}
	if err := m.ParseMultiforecast(testutils.MultiforecastReader(t)); err != nil {
		t.Error(err)
	}
	if err := m.ParseSvgMap(testutils.SvgReader(t)); err != nil {
		t.Error(err)
	}
	return &m
}

const (
	emptyRegexp    = `^$`
	anyRegexp      = `.*`
	coordsRegexp   = `^([\d\.],?)+$`
	instantsRegexp = `morning,afternoon,evening,night`
)

func TestWriteHtml(t *testing.T) {
	appconf.Init([]string{})

	m := testBuildMap(t)

	buf := &bytes.Buffer{}
	err := m.WriteHtml(buf)
	if err != nil {
		t.Errorf("BuildHtml() error: %s", err)
	}
	b, _ := io.ReadAll(buf)
	// display html content
	t.Log(string(b[:400]))
	// TODO: improve html content checks
}

func TestBuildJson(t *testing.T) {
	m := testBuildMap(t)
	j, err := m.BuildJson()
	if err != nil {
		t.Fatalf("BuildJson() error: %s", err)
	}
	// check content
	if j.Name != "France" {
		t.Errorf("jsonMap.Name=%s expected %s", j.Name, "France")
	}
	// TODO improve json content checks
}

func TestForecastURL(t *testing.T) {
	m := testParseHtml(t)

	validationRegexps := map[string]string{
		"bbox":       emptyRegexp,
		"begin_time": emptyRegexp,
		"end_time":   emptyRegexp,
		"time":       emptyRegexp,
		"instants":   instantsRegexp,
		"liste_id":   coordsRegexp,
	}

	u, err := m.ForecastUrl()
	if err != nil {
		t.Fatalf("forecastURL() error: %s", err)
	}
	values := u.Query()
	for key, expr := range validationRegexps {
		re := regexp.MustCompile(expr)
		got, ok := values[key]
		if !ok {
			t.Fatalf("forecastURL() query does not have key '%s'", key)
		}
		if len(got) != 1 {
			t.Fatalf("forecastURL() query ['%s'] has %d values %q, want 1", key, len(got), got)
		}
		if re.Find([]byte(got[0])) == nil {
			t.Errorf("forecastURL() query ['%s']='%s' doesnt match '%s'", key, got[0], expr)
		}
	}
}

const geoRegexp = `^https://meteofrance.com/modules/custom/mf_map_layers_v2/maps/desktop/[A-Z]+/geo_json/[a-z0-9]+-aggrege.json$`

func TestGeographyURL(t *testing.T) {

	m := testParseHtml(t)

	u, err := m.GeographyUrl()
	if err != nil {
		t.Fatalf("geographyURL() error: %s", err)
	}
	expr := regexp.MustCompile(geoRegexp)
	if !expr.Match([]byte(u.String())) {
		t.Errorf("geographyUrl()='%s' does not match '%s'", u.String(), geoRegexp)
	}
}
