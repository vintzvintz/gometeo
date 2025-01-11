package testutils

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

const assets_path = "../test_data/"

const (
	fileHtmlRacine        = "racine.html"
	fileJsonFilterFail    = "json_filter_fail.html"
	fileJsonRacine        = "racine.json"
	fileJsonMultiforecast = "multiforecast.json"
	fileJsonGeography     = "geography.json"
	fileSvgRacine         = "pays007.svg"
)

func HtmlReader(t *testing.T) io.ReadCloser {
	return openFile(t, fileHtmlRacine)
}

func JsonReader(t *testing.T) io.ReadCloser {
	return openFile(t, fileJsonRacine)
}

func MultiforecastReader(t *testing.T) io.ReadCloser {
	return openFile(t, fileJsonMultiforecast)
}

func JsonFailReader(t *testing.T) io.ReadCloser {
	return openFile(t, fileJsonFilterFail)
}

func GeoCollectionReader(t *testing.T) io.ReadCloser {
	return openFile(t, fileJsonGeography)
}

func SvgReader(t *testing.T) io.ReadCloser {
	return openFile(t, fileSvgRacine)
}
func openFile(t *testing.T, name string) io.ReadCloser {
	fp := filepath.Join(assets_path, name)
	f, err := os.Open(fp)
	if err != nil {
		t.Fatalf("os.Open(%s) failed: %v", fp, err)
		return nil
	}
	return f
}
