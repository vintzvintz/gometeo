package mfmap

import (
	"fmt"
	"gometeo/testutils"
	"regexp"
	"testing"
)

func TestParseGeography(t *testing.T) {
	t.Skip("skipped : test files are not up to date")
	j := testutils.GeoCollectionReader(t)
	defer j.Close()

	m := MfMap{
		Data: testParseMap(t),
	}
	err := m.ParseGeography(j)
	if err != nil {
		t.Fatalf("ParseGeography() error: %s", err)
	}
}

func testParseGeoCollection(t *testing.T) *geoCollection {
	j := testutils.GeoCollectionReader(t)
	defer j.Close()

	geo, err := parseGeoCollection(j)
	if err != nil {
		t.Fatal(fmt.Errorf("parseGeoCollection() error: %w", err))
	}
	return geo
}

func TestParseGeoCollection(t *testing.T) {
	geo := testParseGeoCollection(t)
	if geo == nil {
		t.Fatal("parseGeography() returned no data")
	}
}

const (
	geoRegexp = `^https://meteofrance.com/modules/custom/mf_map_layers_v2/maps/desktop/[A-Z]+/geo_json/[a-z0-9]+-aggrege.json$`
	svgRegexp = `^https://meteofrance.com/modules/custom/mf_map_layers_v2/maps/desktop/[A-Z]+/[a-z0-9]+.svg`
)

func TestGeographyQuery(t *testing.T) {

	m := testParseHtml(t)

	t.Run("geographyURL", func(t *testing.T) {
		u, err := m.GeographyURL()
		if err != nil {
			t.Fatalf("geographyURL() error: %s", err)
		}
		expr := regexp.MustCompile(geoRegexp)
		if !expr.Match([]byte(u.String())) {
			t.Errorf("geographyUrl()='%s' does not match '%s'", u.String(), geoRegexp)
		}
	})

	t.Run("svgURL", func(t *testing.T) {
		u, err := m.SvgURL()
		if err != nil {
			t.Fatalf("svgURL() error: %s", err)
		}
		expr := regexp.MustCompile(svgRegexp)
		if !expr.Match([]byte(u.String())) {
			t.Errorf("svgUrl()='%s' does not match '%s'", u.String(), svgRegexp)
		}
	})
}
