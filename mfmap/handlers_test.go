package mfmap_test

import (
	"gometeo/appconf"
	"gometeo/testutils"
	"net/http"
	"net/http/httptest"
	"testing"
)

type testCase struct {
	path       string
	wantStatus int
}

var testsMain = []testCase{
	{"/", http.StatusOK}, // redirection
	{"/france", http.StatusOK},
	{"/france/"+appconf.CacheId()+"/", http.StatusNotFound},
	{"/wesh", http.StatusNotFound},
}

var testsData = []testCase{
	{"/france/data", http.StatusOK},
	{"/france/"+appconf.CacheId()+"data", http.StatusNotFound},
	{"/france/_data", http.StatusNotFound},
}

var testsSvg = []testCase{
	{"/france/svg", http.StatusNotFound},
	{"/france/"+appconf.CacheId()+"/svg", http.StatusOK},
	{"/france/_svg", http.StatusNotFound},
}

func TestMapHandlers(t *testing.T) {

	//setup server on test data
	appconf.Init([]string{})
	m := testBuildMap(t)
	mux := http.NewServeMux()
	m.Register(mux)
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
