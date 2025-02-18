package geojson_test

import (
	"testing"

	_ "gometeo/geojson"
	"gometeo/testutils"
)

func TestParseGeography(t *testing.T) {
	t.Skip("skipped : test files are not up to date")
	j := testutils.GeoCollectionReader(t)
	defer j.Close()

	// m := MfMap{
	// 	Data: testParseMap(t),
	// }
	// err := m.ParseGeography(j)
	// if err != nil {
	// 	t.Fatalf("ParseGeography() error: %s", err)
	// }
}



// func testParseGeography(t *testing.T) *gj.GeoCollection {
// 	j := testutils.GeoCollectionReader(t)
// 	defer j.Close()

// 	m := MfMap{
// 		Data: testParseMap(t),
// 	}
// 	err := m.ParseGeography(j)
// 	if err != nil {
// 		t.Fatalf("ParseGeography() error: %s", err)
// 	}
// 	return m.Geography
// }
