package mfmap

import (
	"fmt"
	"regexp"
	"testing"
)

func TestParseGeography(t *testing.T) {
	file := "geography.json"
	j := openFile(t, file)
	defer j.Close()

	geo, err := parseGeography(j)

	if err != nil {
		t.Fatal(fmt.Errorf("parseGeography() error: %w", err))
	}
	if geo == nil {
		t.Fatal("parseGeography() returned no data")
	}
}

const (
	geoRegexp = `^https://meteofrance.com/modules/custom/mf_map_layers_v2/maps/desktop/[A-Z]+/geo_json/[a-z0-9]+-aggrege.json$`
	svgRegexp = `^https://meteofrance.com/modules/custom/mf_map_layers_v2/maps/desktop/[A-Z]+/[a-z0-9]+.svg`
)

func TestGeographyQuery(t *testing.T) {

	m := parseHtml(t, fileHtmlRacine)

	t.Run("geographyURL", func(t *testing.T) {
		u, err := m.geographyURL()
		if err != nil {
			t.Fatalf("geographyURL() error: %s", err)
		}
		expr := regexp.MustCompile(geoRegexp)
		if !expr.Match([]byte(u.String())) {
			t.Errorf("geographyUrl()='%s' does not match '%s'", u.String(), geoRegexp)
		}
	})

	t.Run("svgURL", func(t *testing.T) {
		u, err := m.svgURL()
		if err != nil {
			t.Fatalf("svgURL() error: %s", err)
		}
		expr := regexp.MustCompile(svgRegexp)
		if !expr.Match([]byte(u.String())) {
			t.Errorf("svgUrl()='%s' does not match '%s'", u.String(), svgRegexp)
		}
	})
}
