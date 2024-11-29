package mfmap

import (
	"fmt"
	"testing"
)

func TestParseMultiforecast(t *testing.T) {

	tests := map[string]struct {
		want interface{}
		got  func(*feature) interface{}
	}{
		"type": {
			want: "Feature",
			got:  func(mf *feature) interface{} { return mf.Type },
		},
	}

	j := openFile(t, "multiforecast.json")
	defer j.Close()

	mf, err := parseMultiforecast(j)
	if err != nil {
		t.Fatal(fmt.Errorf("parseMultiforecast() error: %w", err))
	}
	if mf == nil {
		t.Fatal("parseMultiforecast() returned nil data without error")
	}

	_ = tests
	// TODO: check content
}
