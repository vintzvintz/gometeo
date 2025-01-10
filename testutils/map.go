package testutils

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)


const assets_path = "../test_data/"


func OpenFile(t *testing.T, name string) io.ReadCloser {
	fp := filepath.Join(assets_path, name)
	f, err := os.Open(fp)
	if err != nil {
		t.Fatalf("os.Open(%s) failed: %v", fp, err)
		return nil
	}
	return f
}
