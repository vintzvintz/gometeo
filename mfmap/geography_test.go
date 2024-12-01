package mfmap

import (
	"fmt"
	"testing"
)

const (
	minLat = 35.0
	maxLat = 50.0
	minLng = -5.0
	maxLng = 12.0
)

/*
func validateGeoFeature(t *testing.T, feat *geoFeature) {

	tests := map[string]struct {
		want interface{}
		got  func(*mfFeature) interface{}
	}{
		"root_type": {
			want: "Feature",
			got:  func(f *mfFeature) interface{} { return f.Type },
		},
		"geometry": {
			want: "Point",
			got:  func(f *mfFeature) interface{} { return f.Geometry.Type },
		},
		"country": {
			want: "FR - France",
			got:  func(f *mfFeature) interface{} { return f.Properties.Country },
		},
	}

}
*/

func validateBbox(t *testing.T, bbox *Bbox) {
	latOk := func(lat float64) bool { 
		return (lat > minLat) && (lat < maxLat) 
	}
	lngOk := func(lng float64) bool { 
		return (lng > minLng) && (lng < maxLng) 
	}
	ok := latOk(bbox.Lat1) && latOk(bbox.Lat2) &&
	      lngOk(bbox.Lng1) && lngOk(bbox.Lng2)
	if !ok {
		t.Errorf("out of bound coordinates : %v", bbox)
	}
}

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

	if geo.Type != featureCollectionStr {
		t.Fatalf("parseGeography(%s).Type=%s want %s", file, geo.Type, featureCollectionStr)
	}

	validateBbox(t, geo.Bbox)

	/*
		for _, feat := range geo.Features {
			validateGeoFeature(t, feat)
		}
	*/
}
