package geojson_test

import (
	"fmt"
	"testing"

	"gometeo/testutils"

	gj "gometeo/geojson"
)

func makeMultiforecast(t *testing.T) gj.MultiforecastData {
	j := testutils.MultiforecastReader(t)
	defer j.Close()

	fc, err := gj.ParseMultiforecast(j)
	if err != nil {
		t.Error(fmt.Errorf("parseMfCollection() error: %w", err))
	}
	if len(fc.Features) == 0 {
		t.Fatal("parseMfCollection() returned no data")
	}
	return fc.Features
}

func TestBuildPrevs(t *testing.T) {

	mf := makeMultiforecast(t)
	prevs, err := mf.BuildPrevs()
	if err != nil {
		t.Fatalf("byEcheance error: %s", err)
	}
	if len(prevs) == 0 {
		t.Error("No forecast found in test data")
	}
}

func inspectGraphdata(t *testing.T, g gj.Graphdata) {
	if g == nil {
		t.Fatal("Graphdata is nil")
	}
	var names = []gj.NomSerie{
		gj.Temperature,
		gj.Ressenti,
		gj.WindSpeed,
		gj.WindSpeedGust,
		gj.Iso0,
		gj.CloudCover,
		gj.Hrel,
		gj.Psea,
		gj.Trange,
		gj.Hrange,
		gj.Uv,
	}
	for _, key := range names {
		series, ok := g[key]
		if !ok || len(series) == 0 {
			t.Errorf("missing or empty serie: '%s'", key)
			continue
		}
	}
}

func TestBuildChroniques(t *testing.T) {
	mf := makeMultiforecast(t)
	g, err := mf.BuildChroniques()
	if err != nil {
		t.Fatalf("toChronique() error: %s", err)
	}
	inspectGraphdata(t, g)
}
