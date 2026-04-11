package geojson_test

import (
	"testing"

	gj "gometeo/geojson"
	"gometeo/testutils"
)

func TestParseGeography(t *testing.T) {
	r := testutils.GeoCollectionReader(t)
	defer r.Close()

	subzones := map[string]string{
		"REGIN01": "/meteo-hauts-de-france",
		"REGIN02": "/meteo-normandie",
	}
	gc, err := gj.ParseGeography(r, subzones)
	if err != nil {
		t.Fatalf("ParseGeography() error: %s", err)
	}
	if gc == nil {
		t.Fatal("ParseGeography() returned nil")
	}
	// only subzones present in the subzones map should be kept
	if len(gc.Features) != len(subzones) {
		t.Errorf("got %d features, want %d", len(gc.Features), len(subzones))
	}
	for _, feat := range gc.Features {
		cible := feat.Properties.Prop0.Cible
		wantPath, ok := subzones[cible]
		if !ok {
			t.Errorf("unexpected feature cible %q", cible)
			continue
		}
		if feat.Properties.CustomPath != wantPath {
			t.Errorf("feature %q: CustomPath=%q want %q", cible, feat.Properties.CustomPath, wantPath)
		}
	}
}

func TestParseGeographyEmpty(t *testing.T) {
	r := testutils.GeoCollectionReader(t)
	defer r.Close()

	gc, err := gj.ParseGeography(r, map[string]string{})
	if err != nil {
		t.Fatalf("ParseGeography() error: %s", err)
	}
	if len(gc.Features) != 0 {
		t.Errorf("got %d features, want 0", len(gc.Features))
	}
}
