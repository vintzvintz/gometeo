package crawl

import (
	"bytes"
	/*"io"
	"net/http"
	"net/http/httptest"
	"os"*/
	"strings"
	"testing"
)

var testData01 = MfCache{
	"key_a":        nil,
	"key_b":        {0x00},
	"key_c":        []byte(""),
	"key_d":        []byte("wèèèsh"),
	"":             []byte("Empty key"),
	"unicode_data": []byte(strings.Repeat("Azêrty uiop ", 30)),
	"unicode_kèy":  []byte(strings.Repeat("Azerty uiop ", 30)),
}

func TestCacheHit(t *testing.T) {
	cl := NewClient(testData01)
	for path, expected := range testData01 {
		t.Run(path, func(t *testing.T) {
			got, err := cl.Get(path, CacheOnly)
			if err != nil {
				t.Error(err)
			}
			if !bytes.Equal(got, expected) {
				t.Errorf("got:'%s' expected:'%s'", string(got), string(expected))
			}
		})
	}
}

func TestCacheMiss(t *testing.T) {
	cl := NewClient(nil)
	data, err := cl.Get("key_404", CacheOnly)
	if len(data) > 0 {
		t.Error("MfClient.Get() returned non-empty slice on cache miss")
	}
	if err == nil {
		t.Error("MfClient.Get() returned nil error on cache miss")
	}
}

var pathInvalid = map[string]string{
	"empty string":     "", // empty string does not starts with a slash
	"no_leading_slash": "x",
}

func TestInvalidPath(t *testing.T) {
	for name, path := range pathInvalid {
		t.Run(name, func(t *testing.T) {
			_, err := addUrlBase(path)
			if err == nil {
				t.Errorf("path '%s' not recognized as invalid", path)
			}
		})
	}
}

var pathValid = map[string]struct {
	path     string
	expected string
}{
	"slash":     {path: "/", expected: urlBase + "/"},
	"slashText": {path: "/ressource", expected: urlBase + "/ressource"},
	"base":      {path: urlBase, expected: urlBase},
	"baseSlash": {path: urlBase + "/", expected: urlBase + "/"},
	"baseText":  {path: urlBase + "/ressource", expected: urlBase + "/ressource"},
}

func TestValidPath(t *testing.T) {
	for name, d := range pathValid {
		t.Run(name, func(t *testing.T) {
			got, err := addUrlBase(d.path)
			if err != nil {
				t.Error(err)
			}
			if got != d.expected {
				t.Errorf("got:'%s' expected:'%s", got, d.expected)
			}
		})
	}
}

/*
func TestGet(t *testing.T) {

	name := "test_data/racine.html"
	f, err := os.Open(name)
	if err != nil {
		t.Errorf("%s : %v", name, err)
	}

	// setup mock server
	srv := httptest.NewServer( http.HandlerFunc( func ( w http.ResponseWriter, req *http.Request ) {
		_, err := io.Copy(w, f)
		if err != nil {
			t.Error(err)
		}
	} ))
	defer srv.Close()

//	client := NewClient(nil)
//	client.Get( )


}
*/
