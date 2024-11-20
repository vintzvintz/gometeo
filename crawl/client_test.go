package crawl

import (
	"bytes"
	"io"
	/*
	"net/http"
	"net/http/httptest"
	"os"
	*/
	"strings"
	"testing"
)

var testData01 = MfCache{
	"key_nil":      nil,
	"key_empty":    []byte(""),
	"key_wesh":     []byte("wèèèsh"),
	"":             []byte("Empty key"),
	"unicode_data": []byte(strings.Repeat("Azêrty uiop ", 30)),
	"unicode_kèy":  []byte(strings.Repeat("Azerty uiop ", 30)),
}

func copyCache(src MfCache) (dst MfCache) {
	dst = make(MfCache)
	for k, v := range src {
		vCopy := make([]byte, len(v))
		copy(vCopy, v)
		dst[k] = vCopy
	}
	return
}

func TestCacheHit(t *testing.T) {
	cl := NewClient(copyCache(testData01))
	for path, expected := range testData01 {
		t.Run(path, func(t *testing.T) {
			body, err := cl.Get(path, CacheOnly)
			if err != nil {
				t.Error(err)
			}
			got, _ := io.ReadAll(body)
			if !bytes.Equal(got, expected) {
				t.Errorf("got:'%s' expected:'%s'", string(got), string(expected))
			}
		})
	}
}

func TestCacheMiss(t *testing.T) {
	cl := NewClient(copyCache(testData01))
	_, err := cl.Get("missing_key", CacheOnly)
	if err == nil {
		t.Error("MfClient.Get() returned nil error on cache miss")
	}
}

func TestLookupHit(t *testing.T) {
	c := copyCache(testData01)
	for key, data := range c {
		t.Run(key, func(t *testing.T) {
			r, ok := c.lookup(key)
			if !ok {
				t.Errorf("MfCache.lookup(%s) failed. expected: %s", key, data)
			}
			defer r.Close()
			got, err := io.ReadAll(r)
			if err != nil {
				t.Error(err)
			}
			if !bytes.Equal(got, data) {
				t.Errorf("MfCache.lookup(%s) got '%s' expected '%s'", key, got, data)
			}
		})
	}
}

func TestLookupMiss(t *testing.T) {
	c := copyCache(testData01)
	_, ok := c.lookup("missing_key")
	if ok {
		t.Error("MfCache.lookup() returned nil error on cache miss")
	}
}

// addUrlBase tests
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

func TestUpdaterRead(t *testing.T) {
	cache := MfCache{}
	for k, v := range copyCache(testData01) {
		body := io.NopCloser(bytes.NewReader(v))
		updater := cache.NewCacheUpdater(k, body)

		// lit toutes les données
		_, err := io.ReadAll(updater)
		if err != nil {
			t.Error(err)
		}
		// Close() provoque la mise à jour du cache
		updater.Close()
	}

	// verifie que le cache a capturé toutes les données
	for k, v := range copyCache(testData01) {
		got, ok := cache[k]
		if !ok {
			t.Errorf("Clé %s absente du cache", k)
		}
		if !bytes.Equal(cache[k], v) {
			t.Errorf("MfCache[%s] has wrong data. Got '%s' Expected '%s'", k, got, v)
		}
	}
}


const assets_dir = "../test_data/"
/*
func TestGet(t *testing.T) {

	name := "/racine.html"
	f, err := os.Open(assets_dir + name)
	if err != nil {
		t.Errorf("%s : %v", name, err)
	}

	// setup mock server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		_, err := io.Copy(w, f)
		if err != nil {
			t.Error(err)
		}
	}))
	defer srv.Close()

	client := NewClient(nil)
	body, err := client.Get( "/racine.html", CacheDefault)
	if err != nil {
		t.Errorf("GET ")
	}
}
*/