package mfmap

import (
	"os"
	"testing"
)

func TestParseSvg(t *testing.T) {
	f, err := os.Open("../test_data/pays007.svg")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	m := MfMap{}
	if err := m.ParseSvgMap(f); err != nil {
		t.Fatalf("ParseSvgMap() error: %s", err)
	}
	if len(m.SvgMap) == 0 {
		t.Fatal("ParseSvgMap() produced empty output")
	}
}
