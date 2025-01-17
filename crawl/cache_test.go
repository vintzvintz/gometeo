package crawl

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func dataSet01() map[string][]byte {
	return map[string][]byte{
		"key_nil":      nil,
		"key_empty":    []byte(""),
		"key_wesh":     []byte("wèèèsh"),
		"":             []byte("Empty key"),
		"unicode_data": []byte(strings.Repeat("Azêrty uiop ", 30)),
		"unicode_kèy":  []byte(strings.Repeat("Azerty uiop ", 30)),
	}
}

func TestLookup(t *testing.T) {
	cache := NewCache(dataSet01())

	for key, want := range dataSet01() {
		t.Run(key, func(t *testing.T) {
			r, ok := cache.Lookup(key)
			if !ok {
				t.Fatalf("MfCache.lookup(%s) failed. want %s", key, want)
			}
			defer r.Close()
			got, err := io.ReadAll(r)
			if err != nil {
				t.Fatalf("ReadAll() error: %v", err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("MfCache.lookup(%s) got '%s' expected '%s'", key, got, want)
			}
		})
	}

	key := "missing_key"
	t.Run(key, func(t *testing.T) {
		if _, ok := cache.Lookup(key); ok {
			t.Fatalf("MfCache.lookup(%s) : want error", key)
		}
	})
}

func TestUpdater(t *testing.T) {
	cache := NewCache(dataSet01())
	for key, val := range dataSet01() {
		body := io.NopCloser(bytes.NewReader(val))
		updater := cache.NewUpdater(key, body)

		// read all data through the updater
		_, err := io.ReadAll(updater)
		if err != nil {
			t.Fatal(err)
		}
		updater.Close() // should trigger cache update callback
	}
	// retrieve data from cache and compare with original
	for key, want := range dataSet01() {
		r, ok := cache.Lookup(key)
		if !ok {
			t.Fatalf("Key %s not found in cache", key)
		}
		got, _ := io.ReadAll(r)
		if !bytes.Equal(got, want) {
			t.Fatalf("Cache[%s] got '%s' want '%s'", key, got, want)
		}
	}
}

func TestUpdaterDoubleClose(t *testing.T) {
	c := NewCache(dataSet01())
	data := io.NopCloser(strings.NewReader("data"))
	key := "double_close_test_key"

	t.Run("double close", func(t *testing.T) {
		u := c.NewUpdater(key, data)
		if err := u.Close(); err != nil {
			t.Errorf("cacheUpdater.Close() error on first call :%v", err)
		}
		if err := u.Close(); err != nil {
			t.Errorf("cacheUpdater.Close() error on second call :%v", err)
		}
	})

	t.Run("updated after double close", func(t *testing.T) {
		// is cache properly updated after double close ?
		if _, ok := c.Lookup(key); !ok {
			t.Error("cache not updated after double Close()")
		}
	})
}
