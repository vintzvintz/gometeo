package testutils

import (
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

func RunSvgHandler(
	t *testing.T,
	hdl func(http.ResponseWriter, *http.Request),
	req *http.Request,
	wantStatus int) {
	resp := httptest.NewRecorder()

	//run handler
	hdl(resp, req)

	// check status code
	gotStatus := resp.Result().StatusCode
	if gotStatus != wantStatus {
		t.Fatalf("GET %s wrong response status code. got %d want %d", req.URL, gotStatus, wantStatus)
	}
	if gotStatus != http.StatusOK {
		return
	}

	// check content-type
	wantContentType := "image/svg+xml"
	gotContentType := resp.Result().Header.Get("Content-Type")
	if gotContentType != wantContentType {
		t.Fatalf("GET %s wrong content type. got '%s' want '%s'", req.URL, gotContentType, wantContentType)
	}
	
	// check svg content
	body, err := io.ReadAll(resp.Result().Body)
	if err != nil {
		t.Fatal(err)
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
