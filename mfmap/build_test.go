package mfmap_test

import (
	"bytes"
	"io"
	"testing"
)

func TestWriteHtml(t *testing.T) {
	m := testBuildMap(t)

	buf := &bytes.Buffer{}
	err := m.WriteHtml(buf)
	if err != nil {
		t.Fatalf("BuildHtml() error: %s", err)
	}
	b, _ := io.ReadAll(buf)
	// display html content
	t.Log(string(b[:400]))
	// TODO: improve html content checks
}

func TestBuildJson(t *testing.T) {
	m := testBuildMap(t)
	j, err := m.BuildJson()
	if err != nil {
		t.Fatalf("BuildJson() error: %s", err)
	}
	// check content
	if j.Name != "France" {
		t.Errorf("jsonMap.Name=%s expected %s", j.Name, "France")
	}
	// TODO improve json content checks
}
