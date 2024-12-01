package mfmap

import (
	"fmt"
	"testing"
)

/*
func validateMultiforecastFeature(t *testing.T, feat *mfFeature) {
	tests := map[string]struct {
		want interface{}
		got  func(*mfFeature) interface{}
	}{
		"type": {
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
	if feat == nil {
		t.Fatal("no feature to validate")
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := test.got(feat)
			want := test.want
			if got != want {
				t.Errorf("multiforecast feature %s error: got %s want %s", name, got, want)
			}
		})
	}
}
*/

func TestParseMultiforecast(t *testing.T) {

	j := openFile(t, "multiforecast.json")
	defer j.Close()

	mf, err := parseMultiforecast(j)
	if err != nil {
		t.Fatal(fmt.Errorf("parseMultiforecast() error: %w", err))
	}
	if len(mf) == 0 {
		t.Fatal("parseMultiforecast() returned no data")
	} /*
		for _, feat := range mf {
			validateMultiforecastFeature(t, feat)
		}*/
}
