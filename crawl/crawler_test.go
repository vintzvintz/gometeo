package crawl

import (
	"bytes"
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
		t.Error("MfClient.Get() returned non empty slice on cache miss")
	}
	if err == nil {
		t.Error("MfClient.Get() returned nil error on cache miss")
	}
}
