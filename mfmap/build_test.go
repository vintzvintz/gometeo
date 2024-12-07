package mfmap

import (
	"testing"
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
