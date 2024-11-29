package mfmap

import (
	"fmt"
	"testing"
)

func TestParseMultiforecast(t *testing.T) {

	j := openFile(t, "multiforecast.json")
	defer j.Close()
	m := parseHtml(t, fileHtmlRacine)

	mf, err := m.parseMultiforecast(j)
	if err != nil { 
		t.Fatal( fmt.Errorf("parseMultiforecast() error: %w",err ) )
	}
	if mf == nil {
		t.Fatal("parseMultiforecast() returned nil data without error")
	}

	_ = mf
	// TODO: check content
}