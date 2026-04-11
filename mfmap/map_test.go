package mfmap_test

import (
	"bytes"
	"strings"
	"testing"

	"gometeo/mfmap"
	"gometeo/mfmap/handlers"
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

	m := mfmap.MfMap{Conf: testutils.TestConf}
	if err := m.ParseHtml(f); err != nil {
		t.Fatalf("MfMap.ParseHtml() error: %s", err)
	}
	return &m
}

// buildTestMap returns a JsonMap structure filled form test files

func testBuildMap(t *testing.T) *mfmap.MfMap {
	m := mfmap.MfMap{Conf: testutils.TestConf}
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

func TestWriteHtml(t *testing.T) {
	m := testBuildMap(t)

	buf := &bytes.Buffer{}
	err := handlers.WriteHtml(buf, m)
	if err != nil {
		t.Errorf("BuildHtml() error: %s", err)
	}
	html := buf.String()
	checks := []struct {
		label string
		want  string
	}{
		{"title", "Météo " + m.Name()},
		{"description", "Météo pour la zone " + m.Name()},
		{"path in script", m.Path()},
		{"cacheId", m.Conf.CacheId},
		{"vuejs", m.Conf.VueJs},
	}
	for _, c := range checks {
		if !strings.Contains(html, c.want) {
			t.Errorf("WriteHtml(): %s: %q not found in output", c.label, c.want)
		}
	}
}

func TestBuildJson(t *testing.T) {
	m := testBuildMap(t)
	j, err := handlers.BuildJson(m)
	if err != nil {
		t.Fatalf("BuildJson() error: %s", err)
	}
	if j.Name != "France" {
		t.Errorf("jsonMap.Name=%s expected %s", j.Name, "France")
	}
	if j.Path != m.Path() {
		t.Errorf("jsonMap.Path=%s expected %s", j.Path, m.Path())
	}
	if j.Idtech == "" {
		t.Error("jsonMap.Idtech is empty")
	}
	if j.Taxonomy == "" {
		t.Error("jsonMap.Taxonomy is empty")
	}
	if len(j.SubZones) == 0 {
		t.Error("jsonMap.SubZones is empty")
	}
	if len(j.Prevs) == 0 {
		t.Error("jsonMap.Prevs is empty")
	}
	// PAYS taxonomy: Chroniques should be nil
	if j.Taxonomy == "PAYS" && j.Chroniques != nil {
		t.Error("jsonMap.Chroniques should be nil for PAYS taxonomy")
	}
	// Bbox should have non-zero extent
	bbox := j.Bbox
	if bbox.LngW == bbox.LngE || bbox.LatS == bbox.LatN {
		t.Errorf("jsonMap.Bbox has zero extent: %+v", bbox)
	}
}

func TestMerge(t *testing.T) {
	old := testBuildMap(t)
	old.Schedule.MarkHit()
	old.Schedule.MarkHit()

	newMap := testBuildMap(t)
	newMap.Merge(old, -3, 7)

	if newMap.Schedule.HitCount() != old.Schedule.HitCount() {
		t.Errorf("HitCount after Merge: got %d, want %d", newMap.Schedule.HitCount(), old.Schedule.HitCount())
	}
	if !newMap.Schedule.LastHit().Equal(old.Schedule.LastHit()) {
		t.Errorf("LastHit after Merge: got %v, want %v", newMap.Schedule.LastHit(), old.Schedule.LastHit())
	}
	if len(newMap.Prevs) == 0 {
		t.Error("Prevs is empty after Merge")
	}
}
