package mfmap

import (
	"io"
	"strings"
	"testing"

	"gometeo/testutils"
)

const (
	fileHtmlRacine        = "racine.html"
	fileJsonFilterFail    = "json_filter_fail.html"
	fileJsonRacine        = "racine.json"
	fileJsonMultiforecast = "multiforecast.json"
	fileJsonGeography     = "geography.json"
)

func TestHtmlFilter(t *testing.T) {
	name := fileHtmlRacine
	f := testutils.OpenFile(t, name)
	defer f.Close()

	// get json content
	r, err := htmlFilter(f)
	if err != nil {
		t.Fatalf("jsonFilter error : %s", err)
	}
	_, err = io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to extract JSON data from %s", name)
	}
}

func TestParseHtml(t *testing.T) {
	const expect = "/meteo-france"
	m := parseHtml(t, fileHtmlRacine)
	// check some content
	if m.Data.Info.Path != "/meteo-france" {
		t.Errorf("MfMap.ParseHtml() Info.Path='%s' expected '%s'", m.Data.Info.Path, expect)
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
	m := MfMap{}
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

func parseHtml(t *testing.T, filename string) *MfMap {
	f := testutils.OpenFile(t, filename)
	defer f.Close()

	m := MfMap{}
	if err := m.ParseHtml(f); err != nil {
		t.Fatalf("MfMap.Parse(%s) error: %s", filename, err)
	}
	return &m
}

const apiBaseURL = "https://rpcache-aa.meteofrance.com/internet2018client/2.0"

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

	m := parseHtml(t, fileHtmlRacine)

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			u, err := m.Data.ApiURL(test.path, nil)
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
func buildTestMap(t *testing.T) *MfMap {
	d := testMapParser(t, fileJsonRacine)
	f := testParseMultiforecast(t, fileJsonMultiforecast)
	g := testParseGeoCollection(t, fileJsonGeography)
	_, svgBuf := testCropSVG(t, fileSvgRacine)
	s, err := io.ReadAll(svgBuf)
	if err != nil {
		t.Fatal(err)
	}

	return &MfMap{
		Data:      d,
		Forecasts: f,
		Geography: g,
		SvgMap:    s,
	}
}
