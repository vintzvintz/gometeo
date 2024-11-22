package mfmap

import (
	"io"
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

	// get OS filesize to compare with byte readcount
	fileInfo, err :=  f.Stat()
	if err != nil {
		t.Errorf("os.Stat(%s) failed: %v", name, err)
	}
	filesize := fileInfo.Size()

	// feed raw html
	m, err := NewFrom( f )
	if err != nil {
		t.Errorf("NewFrom(%s) failed: %v", name, err)
	}
	if int64(len(m.html)) != filesize {
		t.Errorf( "NewFrom(%s) : size mismatch len(src)=%d len(buf)=%d", name, filesize, len(m.html) )
	}

	// get json content
	r, err := m.JsonContent()
	if err != nil {
		t.Error(err)
		return
	}
	txt, err := io.ReadAll(io.LimitReader(r,50))
	if err == nil {
		t.Logf( "JSON=%s", txt)
	}
}
