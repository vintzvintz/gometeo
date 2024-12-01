package mfmap

import (
	"fmt"
	"testing"
)

func TestParseMultiforecast(t *testing.T) {

	j := openFile(t, "multiforecast.json")
	defer j.Close()

	mf, err := parseMultiforecast(j)
	if err != nil {
		t.Fatal(fmt.Errorf("parseMultiforecast() error: %w", err))
	}
	if len(mf) == 0 {
		t.Fatal("parseMultiforecast() returned no data")
	} 
}
