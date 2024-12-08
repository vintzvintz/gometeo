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

func TestByEcheances(t *testing.T) {

	mf := testParseMultiforecast(t, fileJsonMultiforecast)
	prevs := mf.ByEcheance()
	if len(prevs) == 0 {
		t.Errorf("No forecast found in %s", fileJsonMultiforecast)
	}
}

func TestGraphData(t *testing.T) {
	mf := testParseMultiforecast(t, fileJsonMultiforecast)
	prevs := mf.ByEcheance()

	d := prevs.toChroniques()
	if (d == nil) || (len(d) == 0) {
		t.Errorf("Graphdata returned nothing from '%s'", fileJsonMultiforecast)
	}
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
