package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"gometeo/mfmap"
	"gometeo/mfmap/handlers"
	"gometeo/testutils"
)

const testCacheId = "testcache"

const assetsPath = "../../test_data/"

type testCase struct {
	path       string
	wantStatus int
}

var testsMain = []testCase{
	{"/", http.StatusOK}, // redirection
	{"/france", http.StatusOK},
	{"/france/" + testCacheId + "/", http.StatusNotFound},
	{"/wesh", http.StatusNotFound},
}

var testsData = []testCase{
	{"/france/data", http.StatusOK},
	{"/france/" + testCacheId + "data", http.StatusNotFound},
	{"/france/_data", http.StatusNotFound},
}

var testsSvg = []testCase{
	{"/france/svg", http.StatusNotFound},
	{"/france/" + testCacheId + "/svg", http.StatusOK},
	{"/france/_svg", http.StatusNotFound},
}

func openFile(t *testing.T, name string) *os.File {
	t.Helper()
	f, err := os.Open(assetsPath + name)
	if err != nil {
		t.Fatalf("os.Open(%s) failed: %v", assetsPath+name, err)
	}
	return f
}

func buildTestMap(t *testing.T) *mfmap.MfMap {
	t.Helper()
	m := mfmap.MfMap{Conf: testutils.TestConf}
	if err := m.ParseHtml(openFile(t, "racine.html")); err != nil {
		t.Fatal(err)
	}
	if err := m.ParseGeography(openFile(t, "geography.json")); err != nil {
		t.Fatal(err)
	}
	if err := m.ParseMultiforecast(openFile(t, "multiforecast.json")); err != nil {
		t.Fatal(err)
	}
	if err := m.ParseSvgMap(openFile(t, "pays007.svg")); err != nil {
		t.Fatal(err)
	}
	return &m
}

func TestMapHandlers(t *testing.T) {
	m := buildTestMap(t)
	mux := http.NewServeMux()
	handlers.Register(mux, m, nil)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	cl := srv.Client()

	t.Run("mainHandler", func(t *testing.T) {
		for _, test := range testsMain {
			testutils.CheckStatusCode(t, cl, srv.URL+test.path, test.wantStatus)
		}
	})

	t.Run("dataHandler", func(t *testing.T) {
		for _, test := range testsData {
			testutils.CheckStatusCode(t, cl, srv.URL+test.path, test.wantStatus)
		}
	})

	t.Run("svgHandler", func(t *testing.T) {
		for _, test := range testsSvg {
			testutils.CheckStatusCode(t, cl, srv.URL+test.path, test.wantStatus)
		}
	})
}
