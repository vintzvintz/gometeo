package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRedirectHtml(t *testing.T) {

	notFound := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}

	h := withOldUrlRedirect(http.HandlerFunc(notFound))

	srv := httptest.NewServer(h)
	defer srv.Close()
	cl := srv.Client()

	// do not follow redirections
	cl.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	type testCase struct {
		path     string
		status   int
		location string
	}
	tests := []testCase{
		{"/france.html", http.StatusMovedPermanently, "/france"},
		{"/francehtml", http.StatusNotFound, ""},
		{"/", http.StatusNotFound, ""},
		{"/html", http.StatusNotFound, ""},
		{"html", http.StatusNotFound, ""},
	}

	for _, test := range tests {
		resp, err := cl.Get(srv.URL + test.path)
		if err != nil {
			t.Error(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != test.status {
			t.Errorf("%s status code %d want %d", test.path, resp.StatusCode, test.status)
		}

		// skip redirection address check if test.location is empty
		if test.location == "" {
			return
		}
		got := resp.Header["Location"][0]
		if got != test.location {
			t.Errorf("%s redirect location %s want %s", test.path, got, test.location)
		}
	}
}
