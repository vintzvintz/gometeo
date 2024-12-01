package mfmap

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
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

func TestHtmlFilter(t *testing.T) {
	name := fileHtmlRacine
	f := openFile(t, name)
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

func TestMapParserFail(t *testing.T) {
	f := openFile(t, fileJsonFilterFail)
	_, err := mapParser(f)
	if err == nil {
		t.Error("error expected")
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

var mapParseTests = map[string]struct {
	want interface{}
	got  func(j *MapData) interface{}
}{
	"Path.BaseUrl": {
		want: "/",
		got:  func(j *MapData) interface{} { return j.Path.BaseUrl },
	},
	"Info.Taxonomy": {
		want: "PAYS",
		got:  func(j *MapData) interface{} { return j.Info.Taxonomy },
	},
	"Info.IdTechnique": {
		want: "PAYS007",
		got:  func(j *MapData) interface{} { return j.Info.IdTechnique },
	},
	"Tools.Config.Site": {
		want: "rpcache-aa",
		got:  func(j *MapData) interface{} { return j.Tools.Config.Site },
	},
	"Tools.Config.BaseUrl": {
		want: "meteofrance.com/internet2018client/2.0",
		got:  func(j *MapData) interface{} { return j.Tools.Config.BaseUrl },
	},
	"ChildrenPOI": {
		want: "VILLE_FRANCE",
		got:  func(j *MapData) interface{} { return j.Children[0].Taxonomy },
	},
	"Subzone": {
		want: SubzoneType{
			Path: "/previsions-meteo-france/auvergne-rhone-alpes/10",
			Name: "Auvergne-Rhône-Alpes",
		},
		got: func(j *MapData) interface{} { return j.Subzones["REGIN10"] },
	},
}

func TestMapParser(t *testing.T) {
	f := openFile(t, fileJsonRacine)
	defer f.Close()

	j, err := mapParser(f)
	if err != nil {
		t.Fatalf("mapParser(%s) error: %v", fileJsonRacine, err)
	}
	for key, test := range mapParseTests {
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

func parseHtml(t *testing.T, filename string) *MfMap {
	f := openFile(t, filename)
	defer f.Close()

	m := MfMap{}
	if err := m.ParseHtml(f); err != nil {
		t.Fatalf("MfMap.Parse(%s) error: %s", filename, err)
	}
	return &m
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
			u, err := m.Data.apiURL(test.path, nil)
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
