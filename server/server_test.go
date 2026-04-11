package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gometeo/content"
	"gometeo/mfmap"
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

func TestHealthzNotReady(t *testing.T) {
	mc := content.New(content.ContentConf{DayMin: -2, DayMax: 2, CacheId: "test"})
	handler := makeMeteoHandler(mc)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("healthz status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}

func TestHealthzReady(t *testing.T) {
	mc := content.New(content.ContentConf{DayMin: -2, DayMax: 2, CacheId: "test"})

	// Feed a map so Ready() returns true
	ch := make(chan *mfmap.MfMap, 1)
	ch <- &mfmap.MfMap{}
	close(ch)
	<-mc.ReceiveMaps(ch)

	handler := makeMeteoHandler(mc)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("healthz status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}
