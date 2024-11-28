package mfmap

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

const assets_path = "../test_data/"

const (
	fileHtmlRacine     = "racine.html"
	fileJsonFilterFail = "json_filter_fail.html"
	fileJsonRacine     = "racine.json"
)

//const apiUrl = "https://rpcache-aa.meteofrance.com/internet2018client/2.0"

func TestJsonFilter(t *testing.T) {
	name := fileHtmlRacine
	f := openFile(t, name)
	defer f.Close()

	// get json content
	r, err := jsonFilter(f)
	if err != nil {
		t.Fatalf("jsonFilter error : %s", err)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to extract JSON data from %s", name)
	}
	data = data[:min(100, len(data))]
	t.Logf("JSON=%s", data)
}

func TestJsonFilterFail(t *testing.T) {
	f := openFile(t, fileJsonFilterFail)
	_, err := jsonFilter(f)
	if err == nil {
		t.Error("JsonReader did not returned error")
		return
	}
}

func openFile(t *testing.T, name string) io.ReadCloser {
	fp := filepath.Join(assets_path, name)
	f, err := os.Open(fp)
	if err != nil {
		t.Fatalf("os.Open(%s) failed: %v", fp, err)
		return nil
	}
	return f
}

var parseTests = map[string]struct {
	want interface{}
	got  func(j *MfMapData) interface{}
}{
	"Path.BaseUrl": {
		want: "/",
		got:  func(j *MfMapData) interface{} { return j.Path.BaseUrl },
	},
	"Info.Taxonomy": {
		want: "PAYS",
		got:  func(j *MfMapData) interface{} { return j.Info.Taxonomy },
	},
	"Info.IdTechnique": {
		want: "PAYS007",
		got:  func(j *MfMapData) interface{} { return j.Info.IdTechnique },
	},
	"Tools.Config.Site": {
		want: "rpcache-aa",
		got:  func(j *MfMapData) interface{} { return j.Tools.Config.Site },
	},
	"Tools.Config.BaseUrl": {
		want: "meteofrance.com/internet2018client/2.0",
		got:  func(j *MfMapData) interface{} { return j.Tools.Config.BaseUrl },
	},
	"ChildrenPOI": {
		want: "VILLE_FRANCE",
		got:  func(j *MfMapData) interface{} { return j.Children[0].Taxonomy },
	},
	"Subzone": {
		want: SubzoneType{
			Path: "/previsions-meteo-france/auvergne-rhone-alpes/10",
			Name: "Auvergne-Rhône-Alpes",
		},
		got: func(j *MfMapData) interface{} { return j.Subzones["REGIN10"] },
	},
}

func TestJsonParser(t *testing.T) {
	f := openFile(t, fileJsonRacine)
	defer f.Close()

	j, err := jsonParser(f)
	if err != nil {
		t.Fatalf("json.Unmarshal(%s) error: %v", fileJsonRacine, err)
	}
	for key, test := range parseTests {
		t.Run(key, func(t *testing.T) {
			got := test.got(j)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("%s got '%s' want '%s'", key, got, test.want)
			}
		})
	}
}

func TestStringFloat(t *testing.T) {
	type Item struct {
		A stringFloat `json:"a"`
		B stringFloat `json:"b"`
		C stringFloat `json:"c"`
		D stringFloat `json:"d"`
	}
	jsonData := []byte(`{"a":null, "c":"51", "d":51}`) // b is missing
	want := Item{A: 0, B: 0, C: 51, D: 51}
	var item Item
	err := json.Unmarshal(jsonData, &item)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(item, want) {
		t.Errorf("stringFloat custom Unmarshall() got %v, want %v", item, want)
	}
}

func parseMapHtml(t *testing.T, filename string) *MfMap {
	f := openFile(t, filename)
	defer f.Close()

	m := MfMap{}
	if err := m.Parse(f); err != nil {
		t.Fatalf("MfMap.Parse(%s) error: %s", filename, err)
	}
	return &m
}

func TestMapParse(t *testing.T) {
	m := parseMapHtml(t, fileHtmlRacine)
	// check some content
	if m.Data.Info.Path != "/meteo-france" {
		t.Errorf("MfMap.Parse() Info.Path='%s' expected '%s'", m.Data.Info.Path, "/meteo-france")
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
			err := m.Parse(r)
			if err == nil {
				t.Error("MfMap.Parse(): error expected")
			}
		})
	}
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

	m := parseMapHtml(t, fileHtmlRacine)

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			u, err := m.Data.apiURL(test.path, nil)
			if err != nil {
				t.Fatalf("ApiURL() error : %s", err)
			}
			got := u.String()
			if got != test.want {
				t.Errorf("forecastUrl()='%s' want '%s'", got, test.want)
			}
		})
	}
}

const (
	emptyRegexp    = `^$`
	anyRegexp      = `.*`
	coordsRegexp   = `^([\d\.],?)+$`
	instantsRegexp = "morning,afternoon,evening,night"
)

func TestForecastQuery(t *testing.T) {
	m := parseMapHtml(t, fileHtmlRacine)

	validationRegexps := map[string]string{
		"bbox":       emptyRegexp,
		"begin_time": emptyRegexp,
		"end_time":   emptyRegexp,
		"time":       emptyRegexp,
		"instants":   instantsRegexp,
		"coords":     coordsRegexp,
	}

	u, err := m.forecastUrl()
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
