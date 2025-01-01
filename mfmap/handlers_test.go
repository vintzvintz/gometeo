package mfmap

import (
	"gometeo/testutils"
	"net/http"
	"net/http/httptest"
	"testing"
)

type testCase struct {
	path       string
	wantStatus int
}

var testsMain = []testCase {
	{"/", http.StatusOK},   // redirection
	{"/france", http.StatusOK},
	{"/wesh", http.StatusNotFound},
}

var testsData = []testCase {
	{"/france/data", http.StatusOK},
	{"/france/_data", http.StatusNotFound},
}

var testsSvg = []testCase {
	{"/france/svg", http.StatusOK},
	{"/france/_svg", http.StatusNotFound},
}

func TestMapHandlers(t *testing.T) {

	//setup server on test data
	m := buildTestMap(t)
	mux := http.ServeMux{}
	m.AddHandlers(&mux)
	srv := httptest.NewServer(&mux)
	defer srv.Close()
	cl := srv.Client()

	t.Run( "mainHandler",func(t *testing.T) { 
		for _, test := range testsMain {
			testutils.CheckStatusCode(t, cl, srv.URL+test.path, test.wantStatus)
		}
	})

	t.Run( "dataHandler",func(t *testing.T) { 
		for _, test := range testsData {
			testutils.CheckStatusCode(t, cl, srv.URL+test.path, test.wantStatus)
		}
	})

	t.Run( "svgHandler",func(t *testing.T) { 
		for _, test := range testsSvg {
			testutils.CheckStatusCode(t, cl, srv.URL+test.path, test.wantStatus)
		}
	})
}
