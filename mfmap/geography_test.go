package mfmap_test

import (
	"gometeo/mfmap"
	"gometeo/testutils"
	"regexp"
	"testing"
)

func TestParseGeography(t *testing.T) {
	t.Skip("skipped : test files are not up to date")
	j := testutils.GeoCollectionReader(t)
	defer j.Close()

	m := mfmap.MfMap{
		Data: testParseMap(t),
	}
	err := m.ParseGeography(j)
	if err != nil {
		t.Fatalf("ParseGeography() error: %s", err)
	}
}

func testParseGeography(t *testing.T) *mfmap.GeoCollection {
	j := testutils.GeoCollectionReader(t)
	defer j.Close()

	m := mfmap.MfMap{
		Data: testParseMap(t),
	}
	err := m.ParseGeography(j)
	if err != nil {
		t.Fatalf("ParseGeography() error: %s", err)
	}
	return m.Geography
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
