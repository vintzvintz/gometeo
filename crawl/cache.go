package crawl

import (
	"bytes"
	"io"
)

type Cache map[string][]byte

// Declaring an enum type for cache control
type CachePolicy int

const (
	CacheDefault  CachePolicy = iota // default. return cached data if available, otherwise send a request
	CacheUpdate                      // ignore cache but store response
	CacheDisabled                    // ignore cache and do not store response
	CacheOnly                        // do not send any request, only cache data only
)

func (m Cache) lookup(path string) (io.ReadCloser, bool) {
	data, ok := m[path]
	return io.NopCloser(bytes.NewReader(data)), ok
}

// cacheUpdater is a io.Reader wrapping a Respose.Body
// to intercept Read() calls and store downloaded content in the cache
type cacheUpdater struct {
	cache  Cache // cache[path] is updated on Close()
	path   string
	body   io.ReadCloser
	buf    []byte
//	closed bool
}

func (c Cache) NewUpdater(path string, body io.ReadCloser) *cacheUpdater {
	return &cacheUpdater{
		cache: c,
		path:  path,
		body:  body,
	}
}

func (cu *cacheUpdater) Read(p []byte) (int, error) {
	nr, err := cu.body.Read(p)
	if err == nil || err == io.EOF {
		cu.buf = append(cu.buf, p[:nr]...)
	}
	return nr, err
}

// Close() updates the cache and close resp.Body()
func (cu *cacheUpdater) Close() error {
//	if cu.closed {
//		return nil
//	}
//	cu.closed = true
	cu.cache[cu.path] = cu.buf
	cu.buf = nil
	cu.body.Close()
	cu.body = nil
	return nil
}
