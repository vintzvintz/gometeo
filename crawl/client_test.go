package crawl

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func dataSet01() *MfCache {
	return &MfCache{
		"key_nil":      nil,
		"key_empty":    []byte(""),
		"key_wesh":     []byte("wèèèsh"),
		"":             []byte("Empty key"),
		"unicode_data": []byte(strings.Repeat("Azêrty uiop ", 30)),
		"unicode_kèy":  []byte(strings.Repeat("Azerty uiop ", 30)),
	}
}

func TestLookupHit(t *testing.T) {
	c := dataSet01()
	for key, data := range *c {
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
	c := dataSet01()
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

var urlBase string = httpsMeteofranceCom

func TestInvalidPath(t *testing.T) {
	for name, path := range pathInvalid {
		t.Run(name, func(t *testing.T) {
			_, err := addUrlBase(path, urlBase)
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
			got, err := addUrlBase(d.path, urlBase)
			if err != nil {
				t.Error(err)
			}
			if got != d.expected {
				t.Errorf("got:'%s' expected:'%s", got, d.expected)
			}
		})
	}
}

func TestUpdater(t *testing.T) {
	cache := MfCache{}
	for k, v := range *dataSet01() {
		body := io.NopCloser(bytes.NewReader(v))
		updater := cache.NewUpdater(k, body)

		// lit toutes les données
		_, err := io.ReadAll(updater)
		if err != nil {
			t.Error(err)
		}
		// Close() provoque la mise à jour du cache
		updater.Close()
	}

	// verifie que le cache a capturé toutes les données
	for k, v := range *dataSet01() {
		got, ok := cache[k]
		if !ok {
			t.Errorf("Clé %s absente du cache", k)
		}
		if !bytes.Equal(cache[k], v) {
			t.Errorf("MfCache[%s] has wrong data. Got '%s' Expected '%s'", k, got, v)
		}
	}
}

func TestUpdaterDoubleClose(t *testing.T) {
	c := MfCache{}
	data := io.NopCloser(strings.NewReader("data"))
	u := c.NewUpdater("key", data)
	if err := u.Close(); err != nil {
		t.Errorf("cacheUpdater.Close() error on first call :%v", err)
	}
	if err := u.Close(); err != nil {
		t.Errorf("cacheUpdater.Close() error on second call :%v", err)
	}
	u = nil // remove reference
	// is cache properly updated after double close ?
	if _, ok := c.lookup("key"); !ok {
		t.Error("cache not properly updated after double Close()")
	}
}

const assets_dir = "../test_data/"

func setupServer(t *testing.T, filename string, cnt *int) (srv *httptest.Server) {
	cookie := &http.Cookie{Name: sessionCookie, Value: "auth_token_string"}
	return setupServerCustom(t, filename, cnt, cookie)
}

/*
	func setupServerNoCookie(t *testing.T, filename string, cnt *int) (srv *httptest.Server) {
		return setupCustomServer(t, filename, cnt, nil)
	}
*/
func setupServerCustom(t *testing.T, filename string, cnt *int, cookie *http.Cookie) (srv *httptest.Server) {

	// prepare data from file
	// empty data if filename is ""
	data := []byte{}
	if filename != "" {
		fp := assets_dir + filename
		f, err := os.Open(fp)
		if err != nil {
			t.Errorf("%s : %v", fp, err)
			return
		}
		data, err = io.ReadAll(f)
		if err != nil {
			t.Errorf("%s : %v", fp, err)
			return
		}
	}
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if cnt != nil {
			*cnt++
		}
		if cookie != nil {
			http.SetCookie(w, cookie)
		}
		_, err := io.Copy(w, bytes.NewReader(data))
		if err != nil {
			t.Error(err)
		}
	}))
	return srv
}

func compareBytesWithFile(t *testing.T, data []byte, filename string) int {
	path := assets_dir + filename
	f, err := os.Open(path)
	if err != nil {
		t.Errorf("os.Open(%s) error: %v", path, err)
		return 1
	}
	defer f.Close()
	file, err := io.ReadAll(f)
	if err != nil {
		t.Error(err)
		return 1
	}
	cmp := bytes.Compare(data, file)
	/*
		if cmp != 0 {
			err := os.WriteFile("fromBytes.html", data, 0660)
			if err != nil {
				t.Error(err)
			}
		}
	*/
	return cmp
}

const fileRacine = "racine.html"

func TestCacheHit(t *testing.T) {
	cl := NewClient()
	cl.cache = dataSet01()
	for path, expected := range *dataSet01() {
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
	cl := NewClient()
	cl.cache = dataSet01()
	_, err := cl.Get("missing_key", CacheOnly)
	if err == nil {
		t.Error("MfClient.Get() returned nil error on cache miss")
	}
}

func TestGetCacheOnly(t *testing.T) {
	var cnt int
	srv := setupServer(t, "", &cnt)
	defer srv.Close()
	client := NewClient()
	client.baseUrl = srv.URL

	for i := 0; i < 2; i++ {
		// Echec attendu en mode cacheOnly car le cache est vide
		data, err := client.Get("/"+fileRacine, CacheOnly)
		_ = data
		if err == nil {
			t.Error("GET should fail in CacheOnly mode")
		}
		if cnt > 0 {
			t.Errorf("GET request sent in CacheOnly mode")
		}
	}
}

func TestGetCacheDefault(t *testing.T) {
	srv := setupServer(t, fileRacine, nil)
	defer srv.Close()
	client := NewClient()
	client.baseUrl = srv.URL

	if htmlFromSrv, err := testClientGet(t, "/"+fileRacine, client, CacheDefault); err != nil {
		if compareBytesWithFile(t, htmlFromSrv, fileRacine) != 0 {
			t.Errorf("différence entre Get(/%s) (fromSrv) et le fichier local '%s'", fileRacine, assets_dir+fileRacine)
		}
	}

	// même requête sans le serveur, devrait réussir avec le cache
	srv.Close()

	if htmlFromCache, err := testClientGet(t, "/"+fileRacine, client, CacheDefault); err != nil {
		// vérifie que la réponse est toujours identique au fichier source
		if compareBytesWithFile(t, htmlFromCache, fileRacine) != 0 {
			t.Errorf("différence entre Get(/%s) (fromCache) et le fichier local '%s'", fileRacine, assets_dir+fileRacine)
		}
	}
}

const initialCachedData = "initial cached data"

func TestGetCacheDisabled(t *testing.T) {

	var cnt int
	srv := setupServer(t, fileRacine, &cnt)
	defer srv.Close()
	client := NewClient()
	client.baseUrl = srv.URL

	// pre-fill cache with data which must not be updated by Get() calls
	client.cache = &MfCache{"/" + fileRacine: []byte(initialCachedData)}

	// perform some requests with CacheDisabled mode
	for i := 0; i < 3; i++ {
		if htmlFromSrv, err := testClientGet(t, "/"+fileRacine, client, CacheDisabled); err != nil {
			if compareBytesWithFile(t, htmlFromSrv, fileRacine) != 0 {
				t.Errorf("différence entre Get(/%s) (fromSrv) et le fichier local '%s'", fileRacine, assets_dir+fileRacine)
			}
		}
		if i+1 != cnt {
			t.Errorf("server should have received %d requests, got %d", i+1, cnt)
		}
	}

	// check cache has not been updated
	cachedData, err := testClientGet(t, "/"+fileRacine, client, CacheOnly)
	if err != nil {
		t.Error(err)
		return
	}
	if !bytes.Equal([]byte(initialCachedData), cachedData) {
		t.Error("cache modified in CacheDisable mode")
	}
}

func TestGetCacheUpdate(t *testing.T) {

	var cnt int
	srv := setupServer(t, fileRacine, &cnt)
	defer srv.Close()
	client := NewClient()
	client.baseUrl = srv.URL

	// pre-fill cache with data to be updated by Get() calls
	client.cache = &MfCache{"/" + fileRacine: []byte(initialCachedData)}

	// send requests with CacheUpdate mode
	if htmlFromSrv, err := testClientGet(t, "/"+fileRacine, client, CacheUpdate); err != nil {
		if compareBytesWithFile(t, htmlFromSrv, fileRacine) != 0 {
			t.Errorf("différence entre Get(/%s) (fromSrv) et le fichier local '%s'", fileRacine, assets_dir+fileRacine)
		}
	}
	if cnt != 1 {
		t.Errorf("server should have received 1 requests, got %d", cnt)
	}

	// check cache has been updated
	cachedData, err := testClientGet(t, "/"+fileRacine, client, CacheOnly)
	if err != nil {
		t.Error(err)
		return
	}
	if compareBytesWithFile(t, cachedData, fileRacine) != 0 {
		t.Errorf("différence entre Get(/%s) (fromCache) et le fichier local '%s'", fileRacine, assets_dir+fileRacine)
	}
}

func testClientGet(t *testing.T, path string, client *MfClient, policy CachePolicy) ([]byte, error) {
	body, err := client.Get(path, policy)
	if err != nil {
		t.Errorf("GET error: %s", err)
		return nil, err
	}
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		t.Error(err)
		return nil, err
	}
	return data, nil
}

func TestGetMissingCookie(t *testing.T) {

	t.Run("MissingCookieError", func(t *testing.T) {
		errMsg := "some error message"
		err := MissingCookieError(errMsg)
		got := err.Error()
		expected := "MissingCookieError: " + errMsg
		if got != expected {
			t.Errorf("MissingCookieError does not generate expected string. got '%s' expected '%s'", got, expected)
		}
	})

	t.Run("MissingCookie", func(t *testing.T) {
		srv := setupServerCustom(t, fileRacine, nil, nil) // no counter, no cookie
		defer srv.Close()
		client := NewClient()
		client.baseUrl = srv.URL

		// send a request, expect a "missing cookie" error
		var err error
		if _, err = client.Get("/"+fileRacine, CacheDisabled); err == nil {
			t.Error("error expected when server does not send auth token")
			return
		}
		if _, ok := err.(MissingCookieError); !ok {
			t.Errorf("MissingCookieError expected, got %v", err)
		}
	})
}

const cookieValA = "cookie_value_A"
const cookieValB = "cookie_value_B"

func assertCookie(t *testing.T, client *MfClient, cookieVal string) {
	cookie := &http.Cookie{Name: sessionCookie, Value: cookieVal}
	srv := setupServerCustom(t, fileRacine, nil, cookie) // no counter
	defer srv.Close()
	client.baseUrl = srv.URL
	if _, err := testClientGet(t, "/"+fileRacine, client, CacheDisabled); err != nil {
		t.Error(err)
		return
	}
	got, _ := Rot13(client.auth_token)
	if got != cookieVal {
		t.Errorf("auth token mismatch. got '%s', expected:'%s'", got, cookieVal)
	}
}

func TestGetModifiedCookie(t *testing.T) {
	client := NewClient()
	assertCookie(t, client, cookieValA)
	assertCookie(t, client, cookieValB)
}
