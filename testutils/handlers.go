package testutils

import (
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

func RunHandler(
	t *testing.T,
	hdl func(http.ResponseWriter, *http.Request),
	req *http.Request,
	wantStatus int) {

	resp := httptest.NewRecorder()
	hdl(resp, req)
	body, err := io.ReadAll(resp.Result().Body)
	if err != nil {
		t.Fatal(err)
	}
	gotStatus := resp.Result().StatusCode
	if gotStatus != wantStatus {
		t.Fatalf("GET %s wrong response status code. got %d want %d", req.URL, gotStatus, wantStatus)
	}
	if len(body) == 0 {
		return
	}
	re := regexp.MustCompile(`<svg`)
	if !re.Match(body) {
		t.Fatalf("missing <svg> tag")
	}
}

func CheckStatusCode(t *testing.T, cl *http.Client, path string, wantStatus int) {
	resp, err := cl.Get(path)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != wantStatus {
		t.Fatalf("%s returned status code %d, expect %d", path, resp.StatusCode, wantStatus)
	}
}
