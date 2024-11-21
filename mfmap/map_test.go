package mfmap

import (
	"os"
	"testing"
)

const assets_path = "../test_data/"

func TestNewMap(t *testing.T) {

	const name = assets_path + "racine.html"
	f, err := os.Open(name)
	if err != nil {
		t.Errorf("os.Open(%s) failed: %v", name, err)
	}
	defer f.Close()

	fileInfo, err :=  f.Stat()
	if err != nil {
		t.Errorf("os.Stat(%s) failed: %v", name, err)
	}

	var m *MfMap
	nb, err := m.ReadFrom( f )
	if err != nil {
		t.Errorf("MfMap.ReadFrom(%s) failed: %v", name, err)
	}
	filesize := fileInfo.Size()
	if nb != filesize {
		t.Errorf("MfMap.ReadFrom(%s) returned %d, expected %d from fileInfo.Size()", name, nb, filesize)
	}
	// TODO: check parse results
}
