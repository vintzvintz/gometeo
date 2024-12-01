package mfmap

import (
	"fmt"
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
