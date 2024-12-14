package mfmap

import (
	"fmt"
	"testing"
	"time"
)

func TestBuildJson(t *testing.T) {
	m := buildTestMap(t)
	j, err := m.BuildJson()
	if err != nil {
		t.Fatalf("BuildJson() error: %s", err)
	}
	// check content
	if j.Name != "France" {
		t.Errorf("jsonMap.Name=%s expected %s", j.Name, "France")
	}

	/*
		type JsonMap struct {
			Name string
			Idtech string
			Taxonomy string
			SubZones geoFeatures
			Bbox Bbox
		}
	*/
}

func TestBuildGraphdata(t *testing.T) {
	m := buildTestMap(t)
	g, err := m.BuildGraphdata()
	if err != nil {
		t.Fatalf("BuildGraphdata() error: %s", err)
	}
	inspectGraphdata(t, g)
}

func TestByEcheances(t *testing.T) {

	mf := testParseMultiforecast(t, fileJsonMultiforecast)
	prevs := mf.ByEcheance()
	if len(prevs) == 0 {
		t.Errorf("No forecast found in %s", fileJsonMultiforecast)
	}
}

func inspectGraphdata(t *testing.T, g Graphdata) {
	if g == nil {
		t.Fatal("Graphdata is nil")
	}
	for _, key := range append(forecastsChroniques, dailiesChroniques...) {
		series, ok := g[key]
		if !ok || len(series) == 0 {
			t.Errorf("missing or empty serie: '%s'", key)
			continue
		}
	}
}

func TestToChronique(t *testing.T) {
	mf := testParseMultiforecast(t, fileJsonMultiforecast)
	g, err := mf.toChroniques()
	if err != nil {
		t.Fatalf("toChronique() error: %s", err)
	}
	inspectGraphdata(t, g)
}

func TestEcheanceString(t *testing.T) {

	m := morningStr
	d, _ := time.Parse(time.RFC3339, "2024-12-02T15:51:12.000Z")
	e := Echeance{MomentName(m), d}
	want := fmt.Sprintf("%4d-%02d-%02d %s", d.Year(), d.Month(), d.Day(), m)

	got := e.String()
	if got != want {
		t.Errorf("Echeance.String()='%s' want '%s'", got, want)
	}
}

func TestFindDaily(t *testing.T) {
	mf := testParseMultiforecast(t, fileJsonMultiforecast)
	id := CodeInsee("440360") // "name": "Châteaubriant"
	ech, err := time.Parse(time.RFC3339, "2024-12-02T00:00:00.000Z")
	if err != nil {
		t.Fatal(err)
	}
	d := mf.FindDaily(id, ech)
	if d == nil {
		t.Fatalf("FindDaily() did not found daily forecast for location '%s' at '%s'", id, ech)
	}
}

func TestBuildHtml(t *testing.T) {
	m := buildTestMap(t)
	h, err := m.BuildHtml()
	if err != nil {
		t.Fatalf("BuildHtml() error: %s", err)
	}
	// check html content
	// TODO
	_ = h
}
