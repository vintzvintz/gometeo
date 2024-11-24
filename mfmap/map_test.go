package mfmap

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const assets_path = "../test_data/"

const (
	fileRacine         = "racine.html"
	fileJsonFilterFail = "json_filter_fail.html"
)

const apiUrl = "https://rpcache-aa.meteofrance.com/internet2018client/2.0"

func TestJsonFilter(t *testing.T) {
	name := fileRacine
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

func openFile(t *testing.T, name string) io.ReadCloser {
	fp := filepath.Join(assets_path, name)
	f, err := os.Open(fp)
	if err != nil {
		t.Fatalf("os.Open(%s) failed: %v", fp, err)
		return nil
	}
	return f
}

func TestJsonFilterFail(t *testing.T) {
	f := openFile(t, fileJsonFilterFail)
	_, err := jsonFilter(f)
	if err == nil {
		t.Error("JsonReader did not returned error")
		return
	}
}

func TestJsonParser(t *testing.T) {
	const name = assets_path + "racine.json"
	f, err := os.Open(name)
	if err != nil {
		t.Errorf("os.Open(%s) failed: %v", name, err)
		return
	}
	defer f.Close()

	j, err := jsonParser(f)
	if err != nil {
		t.Errorf("json.Unmarshal(%s) failed: %v", name, err)
	}
	t.Log(j)
	t.Run("basic", func(t *testing.T) {
		// check few leafs
		if j.Path.BaseUrl != "/" {
			t.Errorf("j.Path.BaseUrl=%v expected /", j.Path.BaseUrl)
		}
		if j.Info.Taxonomy != "PAYS" {
			t.Errorf("j.MapLayersV2.Taxonomy=%v expected FRANCE", j.Info.Taxonomy)
		}
		if j.Info.IdTechnique != "PAYS007" {
			t.Errorf("j.MapLayersV2.IdTechnique=%v expected PAYS007", j.Info.IdTechnique)
		}
	})

	t.Run("config", func(t *testing.T) {
		got := j.ApiURL()
		if got != apiUrl {
			t.Errorf("ApiUrl() got %s expected %s", got, apiUrl)
		}
	})

	t.Run("children", func(t *testing.T) {
		firstChild := j.Children[0]
		got := firstChild.Taxonomy
		expected := "VILLE_FRANCE"
		if got != expected {
			t.Errorf("MapChildren() taxonomy got %s expected %s)", got, expected)
		}
	})

	t.Run("subzones", func(t *testing.T) {
		id := "REGIN10" // auvergne rhone alpes

		expected := struct{ Path, Name string }{
			Path: "/previsions-meteo-france/auvergne-rhone-alpes/10",
			Name: "Auvergne-Rhône-Alpes",
		}
		sz, ok := j.Subzones[id]
		if !ok {
			t.Errorf("subzone %s not found", id)
		}
		if sz.Name != expected.Name {
			t.Errorf("subzone[%s].name got '%s', expected '%s'", id, sz.Name, expected.Name)
		}
		if sz.Path != expected.Path {
			t.Errorf("subzone[%s].name got '%s', expected '%s'", id, sz.Path, expected.Path)
		}
	})
}

func TestMapParse(t *testing.T) {
	const name = assets_path + "racine.html"
	f, err := os.Open(name)
	if err != nil {
		t.Errorf("os.Open(%s) failed: %v", name, err)
		return
	}
	defer f.Close()

	m := MfMap{}
	if err = m.Parse(f); err != nil {
		t.Errorf("parse error: %s", err)
	}
	// check content
	if m.Data.Info.Path != "/meteo-france" {
		t.Errorf("MfMap.Parse() Info.Path='%s' expected '%s'", m.Data.Info.Path, "/meteo-france")
	}
}

func TestMapParseFail(t *testing.T) {

	tests := map[string]string{
		"missingJson": `
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
 <script type="application/json" data-drupal-selector="drupal-settings-json">{invalid json content}</script>
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

func TestFetchPrevs(t *testing.T) {

}
