package testutils

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gometeo/mfmap"
	"gometeo/mfmap/schedule"
)

const assets_path = "../test_data/"

const (
	fileHtmlRacine        = "racine.html"
	fileJsonFilterFail    = "json_filter_fail.html"
	fileJsonMultiforecast = "multiforecast.json"
	fileJsonGeography     = "geography.json"
	fileSvgRacine         = "pays007.svg"
)

var TestConf = mfmap.MapConf{
	CacheId:  "testcache",
	VueJs:    "vue.esm-browser.dev.js",
	Upstream: "https://meteofrance.com",
	Rates: schedule.UpdateRates{
		HotDuration: 72 * time.Hour,
		HotMaxAge:   60 * time.Minute,
		ColdMaxAge:  240 * time.Minute,
	},
}

func BuildTestMap(t *testing.T) *mfmap.MfMap {
	m := mfmap.MfMap{Conf: TestConf}
	if err := m.ParseHtml(openFile(t, fileHtmlRacine)); err != nil {
		t.Error(err)
	}
	if err := m.ParseGeography(openFile(t, fileJsonGeography)); err != nil {
		t.Error(err)
	}
	if err := m.ParseMultiforecast(openFile(t, fileJsonMultiforecast)); err != nil {
		t.Error(err)
	}
	if err := m.ParseSvgMap(openFile(t, fileSvgRacine)); err != nil {
		t.Error(err)
	}
	return &m
}

func HtmlReader(t *testing.T) io.ReadCloser {
	return openFile(t, fileHtmlRacine)
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
