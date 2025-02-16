package mfmap_test

import (
	"strings"
	"testing"

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
func testBuildMap(t *testing.T) *mfmap.MfMap {
	return &mfmap.MfMap{
		Data:      testParseMap(t),
		Multi:     testParseMultiforecast(t),
		Geography: testParseGeography(t),
		SvgMap:    testParseSvg(t),
	}
}
