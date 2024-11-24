package mfmap

import (
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
	got  func(j *JsonData) interface{}
}{
	"Path.BaseUrl": {
		want: "/",
		got:  func(j *JsonData) interface{} { return j.Path.BaseUrl },
	},
	"Info.Taxonomy": {
		want: "PAYS",
		got:  func(j *JsonData) interface{} { return j.Info.Taxonomy },
	},
	"Info.IdTechnique": {
		want: "PAYS007",
		got:  func(j *JsonData) interface{} { return j.Info.IdTechnique },
	},
	"Tools.Config.Site": {
		want: "rpcache-aa",
		got:  func(j *JsonData) interface{} { return j.Tools.Config.Site },
	},
	"Tools.Config.BaseUrl": {
		want: "meteofrance.com/internet2018client/2.0",
		got:  func(j *JsonData) interface{} { return j.Tools.Config.BaseUrl },
	},
	"ChildrenPOI": {
		want: "VILLE_FRANCE",
		got:  func(j *JsonData) interface{} { return j.Children[0].Taxonomy },
	},
	"Subzone": {
		want: SubzoneType{
			Path: "/previsions-meteo-france/auvergne-rhone-alpes/10",
			Name: "Auvergne-Rhône-Alpes",
		},
		got: func(j *JsonData) interface{} { return j.Subzones["REGIN10"] },
	},
	"api_URL": {
		want: "https://rpcache-aa.meteofrance.com/internet2018client/2.0",
		got:  func(j *JsonData) interface{} { return j.ApiURL() },
	},
}

func TestJsonParser(t *testing.T) {
	f := openFile(t, fileJsonRacine)
	defer f.Close()

	j, err := jsonParser(f)
	if err != nil {
		t.Errorf("json.Unmarshal(%s) error: %v", fileJsonRacine, err)
	}
	//t.Log(j)

	for key, test := range parseTests {
		t.Run(key, func(t *testing.T) {
			got := test.got(j)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("%s got '%s' want '%s'", key, got, test.want)
			}
		})
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

const multiforecastUrl = "https://rpcache-aa.meteofrance.com/internet2018client/2.0/multiforecast"

func TestForecastUrl(t *testing.T) {
	m := parseMapHtml(t, fileHtmlRacine)

	want := multiforecastUrl
	got := m.forecastUrl()
	if got != want {
		t.Errorf("forecastUrl()='%s' want '%s'", got, want)
	}
}

func TestForecastQuery(t *testing.T) {
	m := parseMapHtml(t, fileHtmlRacine)

	want := "wesh"
	got := m.forecastQuery()
	if got != want {
		t.Errorf("forecastQuery()='%s' want '%s'", got, want)
	}
}
