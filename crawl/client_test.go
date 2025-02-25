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

const assets_dir = "../test_data/"

// addUrlBase tests
var urlBaseTest string = "https://example.com"

func TestAddUrlBase(t *testing.T) {

	var pathInvalid = map[string]string{
		"empty string":     "", // empty string does not starts with a slash
		"no_leading_slash": "x",
	}
	var pathValid = map[string]struct {
		path     string
		expected string
	}{
		"slash":     {path: "/", expected: urlBaseTest + "/"},
		"slashText": {path: "/path", expected: urlBaseTest + "/path"},
		"base":      {path: urlBaseTest, expected: urlBaseTest},
		"baseSlash": {path: urlBaseTest + "/", expected: urlBaseTest + "/"},
		"baseText":  {path: urlBaseTest + "/path", expected: urlBaseTest + "/path"},
	}

	t.Run("empty baseUrl", func(t *testing.T) {
		cl := NewClient("")
		_, err := cl.addUrlBase("/")
		if err == nil {
			t.Errorf("expect error on empty urlBase")
		}
	})

	t.Run("invalid paths", func(t *testing.T) {
		for name, path := range pathInvalid {
			t.Run(name, func(t *testing.T) {
				cl := NewClient(urlBaseTest)
				_, err := cl.addUrlBase(path)
				if err == nil {
					t.Errorf("expect error on invalid path '%s'", path)
				}
			})
		}
	})

	t.Run("valid paths", func(t *testing.T) {
		for name, d := range pathValid {
			t.Run(name, func(t *testing.T) {
				cl := NewClient(urlBaseTest)
				got, err := cl.addUrlBase(d.path)
				if err != nil {
					t.Error(err)
				}
				if got != d.expected {
					t.Errorf("got:'%s' expected:'%s", got, d.expected)
				}
			})
		}
	})
}

func setupServerAndClient(t *testing.T, filename string, cnt *int) (srv *httptest.Server, client *Client) {
	cookie := &http.Cookie{Name: sessionCookie, Value: "random_auth_token_value"}
	srv = setupServerCustom(t, filename, cnt, cookie)
	client = NewClient(srv.URL)
	return
}

func setupServerCustom(t *testing.T, filename string, cnt *int, cookie *http.Cookie) (srv *httptest.Server) {
	// prepare data from file
	data := []byte{}
	if filename != "" {
		fp := assets_dir + filename
		f, err := os.Open(fp)
		if err != nil {
			t.Fatalf("%s : %v", fp, err)
		}
		data, err = io.ReadAll(f)
		if err != nil {
			t.Fatalf("%s : %v", fp, err)
		}
	}

	// start an http server replying with data to any request
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if cnt != nil {
			*cnt++
		}
		if cookie != nil {
			http.SetCookie(w, cookie)
		}
		_, err := io.Copy(w, bytes.NewReader(data))
		if err != nil {
			t.Fatalf("io.Copy() error: %s", err)
		}
	}))
	return srv
}

// setupServerWithStatus starts an http server replying with empty body and provided status code
func setupServerWithStatus(t *testing.T, status int) *httptest.Server {
	_ = t
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(status)
	}))
	return srv
}

func assertGetEqualsFile(t *testing.T, client *Client, filename string, policy CachePolicy) {
	t.Helper()
	path := assets_dir + filename
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("os.Open(%s) error: %v", path, err)
	}
	defer f.Close()
	assertGetEqualsBytes(t, client, "/"+filename, f, policy)
}

func assertGetEqualsBytes(t *testing.T, client *Client, path string, r io.Reader, policy CachePolicy) {
	t.Helper()
	want, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("io.Readall(%s) error: %v", path, err)
	}
	got := testClientGet(t, client, path, policy)
	if !bytes.Equal(got, want) {
		got = got[:min(200, len(got))] // truncate in error message
		want = want[:min(200, len(want))]
		t.Fatalf("client.Get(%s) \n\tgot(%s) \n\twant '%s'", path, got, want)
	}
}

const fileRacine = "racine.html"

func TestCacheHit(t *testing.T) {
	cl := NewClient("")
	cl.cache = NewCache(dataSet01())
	for path, expected := range dataSet01() {
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
	key := "missing_key"
	cl := NewClient("")
	cl.cache = NewCache(dataSet01())
	cl.client = nil
	_, err := cl.Get(key, CacheOnly)
	if err == nil {
		t.Errorf("MfClient.Get() should have returned error on '%s'", key)
	}
}

func TestGetCacheOnly(t *testing.T) {
	var cnt int
	srv, client := setupServerAndClient(t, "", &cnt)
	defer srv.Close()

	for i := 0; i < 2; i++ {
		// Echec attendu en mode cacheOnly car le cache est vide
		data, err := client.Get("/", CacheOnly)
		_ = data
		if err == nil {
			t.Error("GET should fail in CacheOnly mode")
		}
		if cnt > 0 {
			t.Errorf("GET request performed in CacheOnly mode")
		}
	}
}

func TestGetCacheDefault(t *testing.T) {
	file := fileRacine
	srv, client := setupServerAndClient(t, file, nil)
	defer srv.Close()

	// second request should succeed and return data from cache without requesting server
	assertGetEqualsFile(t, client, file, CacheDefault)
	srv.Close()
	assertGetEqualsFile(t, client, file, CacheDefault)
}

const initialCachedData = "initial cached data"

func TestGetCacheDisabled(t *testing.T) {
	const file = fileRacine
	path := "/" + file
	var cnt int
	srv, client := setupServerAndClient(t, file, &cnt)
	defer srv.Close()
	// pre-fill cache with data which must not be updated by Get() calls
	client.cache = NewCache(map[string][]byte{
		path: []byte(initialCachedData),
	})

	const nbReq = 3
	t.Run("repeated requests", func(t *testing.T) {
		for i := 0; i < nbReq; i++ {
			assertGetEqualsFile(t, client, file, CacheDisabled)
			if i+1 != cnt {
				t.Errorf("server should have received %d requests, got %d", i+1, cnt)
			}
		}
	})
	srv.Close()

	// check cache has not been updated
	assertGetEqualsBytes(t, client, path, strings.NewReader(initialCachedData), CacheOnly)
}

func TestGetCacheUpdate(t *testing.T) {
	var file string = fileRacine
	var path string = "/" + file
	var cnt int
	srv, client := setupServerAndClient(t, file, &cnt)
	defer srv.Close()

	// pre-fill cache with data to be updated by Get() calls
	client.cache = NewCache(map[string][]byte{
		path: []byte(initialCachedData),
	})

	// send requests with CacheUpdate mode
	assertGetEqualsFile(t, client, file, CacheUpdate)
	if cnt != 1 {
		t.Errorf("server should have received 1 requests, got %d", cnt)
	}

	// check cache has been updated
	cnt = 0 // reset counter
	assertGetEqualsFile(t, client, file, CacheOnly)
	if cnt > 0 {
		t.Errorf("server got %d requests, expected 0", cnt)
	}
}

func testClientGet(t *testing.T, client *Client, path string, policy CachePolicy) []byte {
	t.Helper()
	body, err := client.Get(path, policy)
	if err != nil {
		t.Fatalf("client.Get() error: %s", err)
	}
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("ReadAll() error: %v", err)
	}
	return data
}

func TestGetMissingCookie(t *testing.T) {

	t.Run("MissingCookie type", func(t *testing.T) {
		errMsg := "some error message"
		err := MissingCookieError(errMsg)
		got := err.Error()
		expected := "MissingCookieError: " + errMsg
		if got != expected {
			t.Errorf("MissingCookieError does not generate expected string. got '%s' expected '%s'", got, expected)
		}
	})

	t.Run("missing cookie error", func(t *testing.T) {
		srv := setupServerCustom(t, "", nil, nil) // no data, no counter, no cookie
		defer srv.Close()
		client := NewClient(srv.URL)

		// send a request, expect a "missing cookie" error
		var err error
		if _, err = client.Get("/", CacheDisabled); err == nil {
			t.Error("error expected when server does not send auth token")
			return
		}
		if _, ok := err.(MissingCookieError); !ok {
			t.Errorf("MissingCookieError expected, got %v", err)
		}
	})
}

func assertCookie(t *testing.T, client *Client, cookieVal string) {
	t.Helper()
	cookie := &http.Cookie{Name: sessionCookie, Value: cookieVal}
	srv := setupServerCustom(t, fileRacine, nil, cookie) // no counter
	defer srv.Close()
	client.baseUrl = srv.URL // point existing client to test server

	// perform request just to get the session cookie
	_ = testClientGet(t, client, "/", CacheDisabled)

	// check if cookie has expected value
	got, _ := Rot13(client.token.Get())
	if got != cookieVal {
		t.Errorf("auth token mismatch. got '%s', expected:'%s'", got, cookieVal)
	}
}

func TestGetModifiedCookie(t *testing.T) {
	const (
		cookieValA = "cookie_value_A"
		cookieValB = "cookie_value_B"
	)
	client := NewClient("")
	assertCookie(t, client, cookieValA)
	assertCookie(t, client, cookieValB)
}

// test invalid paths
func TestGetBadPath(t *testing.T) {
	badPaths := map[string]string{
		"emptyString":    "", // empty string does not starts with a slash
		"noLeadingSlash": "x",
		"invalid scheme": "://example.com/",
	}
	client := NewClient("")
	client.client = nil // should prevent real requests
	for name, path := range badPaths {
		t.Run(name, func(t *testing.T) {
			_, err := client.Get(path, CacheDisabled)
			if err == nil {
				t.Errorf("expected error on invalid path '%s'", path)
			}
		})
	}
}

func TestHttpErrors(t *testing.T) {
	client := NewClient("")
	statusCodes := []int{401, 404, 500}

	for _, code := range statusCodes {
		t.Run(http.StatusText(code), func(t *testing.T) {
			srv := setupServerWithStatus(t, code)
			defer srv.Close()
			client.baseUrl = srv.URL
			_, err := client.Get("/", CacheDefault)
			if err == nil {
				t.Fatalf("client.Get() did not returned an error")
			}
		})
	}
}
