package mfmap

import (
	"io"
	"os"
	"strings"
	"testing"
)

const assets_path = "../test_data/"

const apiUrl = "https://rpcache-aa.meteofrance.com/internet2018client/2.0"

/*
func TestNewMap(t *testing.T) {

	const name = assets_path + "racine.html"
	f, err := os.Open(name)
	if err != nil {
		t.Errorf("os.Open(%s) failed: %v", name, err)
		return
	}
	defer f.Close()

	// get OS filesize to compare with byte readcount
	fileInfo, err :=  f.Stat()
	if err != nil {
		t.Errorf("os.Stat(%s) failed: %v", name, err)
	}
	filesize := fileInfo.Size()

	// feed raw html
	m, err := NewFrom( f )
	if err != nil {
		t.Errorf("NewFrom(%s) failed: %v", name, err)
	}
	if int64(len(m.html)) != filesize {
		t.Errorf( "NewFrom(%s) : size mismatch len(src)=%d len(buf)=%d", name, filesize, len(m.html) )
	}

	// get json content
	r, err := m.JsonContent()
	if err != nil {
		t.Error(err)
		return
	}
	txt, err := io.ReadAll(io.LimitReader(r,50))
	if err == nil {
		t.Logf( "JSON=%s", txt)
	}
}
*/

func TestJsonFilter(t *testing.T) {
	const name = assets_path + "racine.html"
	f, err := os.Open(name)
	if err != nil {
		t.Errorf("os.Open(%s) failed: %v", name, err)
		return
	}
	defer f.Close()

	// get json content
	r, err := jsonFilter(f)
	if err != nil {
		t.Error(err)
		return
	}
	data, err := io.ReadAll(io.LimitReader(r, 50))
	if err != nil {
		t.Errorf("Failed to extract JSON data from %s", name)
		return
	}
	t.Logf("JSON=%s", data)
}

func TestJsonFilterFail(t *testing.T) {
	r := strings.NewReader(`
<html>
<head>
<title>JsonReader test</head>
<body>
<script> 
	script element without attributes
</script>
<script src="/wesh.js" data-drupal-selector="drupal-settings-json">
	wrong src attr, missing type attr
/script>
<script type="application/json" data-drupal-selector="drupal-settings-json">
	wrong type attr
/script>
<script type="application/json" data-drupal-selector="wesh">
	wrong drupal attr
</script>
<body>
</html>`)
	_, err := jsonFilter(r)
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
			t.Log(err)
		})
	}
}
