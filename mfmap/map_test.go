package mfmap

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"reflect"
)

const assets_path = "../test_data/"

const (
	fileHtmlRacine     = "racine.html"
	fileJsonFilterFail = "json_filter_fail.html"
	fileJsonRacine     = "racine.json"
)

const apiUrl = "https://rpcache-aa.meteofrance.com/internet2018client/2.0"

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

func TestJsonParser(t *testing.T) {
	f := openFile(t, fileJsonRacine)
	defer f.Close()

	j, err := jsonParser(f)
	if err != nil {
		t.Errorf("json.Unmarshal(%s) error: %v", fileJsonRacine, err)
	}
	//t.Log(j)
	t.Run("configuration data", func(t *testing.T) {
		// check few leafs
		if j.Path.BaseUrl != "/" {
			t.Errorf("j.Path.BaseUrl=%v expected /", j.Path.BaseUrl)
		}
		if j.Info.Taxonomy != "PAYS" {
			t.Errorf("j.Info.Taxonomy=%v expected FRANCE", j.Info.Taxonomy)
		}
		if j.Info.IdTechnique != "PAYS007" {
			t.Errorf("j.Info.IdTechnique=%v expected PAYS007", j.Info.IdTechnique)
		}
	})

	t.Run("apiUrl", func(t *testing.T) {
		got := j.ApiURL()
		if got != apiUrl {
			t.Errorf("ApiUrl() got %s expected %s", got, apiUrl)
		}
	})

	t.Run("children poi", func(t *testing.T) {
		firstChild := j.Children[0]
		got := firstChild.Taxonomy
		expected := "VILLE_FRANCE"
		if got != expected {
			t.Errorf("MapChildren() taxonomy got %s expected %s)", got, expected)
		}
	})

	t.Run("subzones", func(t *testing.T) {
		id := "REGIN10" // auvergne rhone alpes
		expected := SubzoneType{
			Path: "/previsions-meteo-france/auvergne-rhone-alpes/10",
			Name: "Auvergne-Rhône-Alpes",
		}
		got, ok := j.Subzones[id]
		if !ok {
			t.Fatalf("subzone %s not found", id)
		}
		if !reflect.DeepEqual(got, expected) {
			t.Errorf("subzone[%s]='%s', expected '%s'", id, got, expected)
		}
	})
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

func TestPrevsReq(t *testing.T) {

//	req := 

}
