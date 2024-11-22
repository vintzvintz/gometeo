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
		return
	}
	defer f.Close()

	fileInfo, err :=  f.Stat()
	if err != nil {
		t.Errorf("os.Stat(%s) failed: %v", name, err)
	}
	filesize := fileInfo.Size()

	
	m, err := NewFrom( f )

//	nb, err := m.ReadFrom( f )
	if err != nil {
		t.Errorf("NewFrom(%s) failed: %v", name, err)
	}
	/*
	filesize := fileInfo.Size()
	if nb != filesize {
		t.Errorf("MfMap.ReadFrom(%s) returned %d, expected %d from fileInfo.Size()", name, nb, filesize)
	}
		*/
	// TODO: check parse results

	if int64(len(m.html)) != filesize {
		t.Errorf( "NewFrom(%s) : size mismatch len(src)=%d len(buf)=%d", name, filesize, len(m.html) )
	}

	_ = filesize
}
