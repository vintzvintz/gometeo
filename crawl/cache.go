package crawl

import (
	"bytes"
	"io"
	"sync"
)

type Cache struct {
	data  map[string][]byte
	mutex sync.Mutex
}

func NewCache(data map[string][]byte) *Cache {
	if data == nil {
		data = make(map[string][]byte)
	}
	return &Cache{
		data: data,
	}
}

func (c *Cache) Lookup(path string) (io.ReadCloser, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	data, ok := c.data[path]
	if !ok {
		return nil, false
	}
	return io.NopCloser(bytes.NewReader(data)), ok
}

func (c *Cache) Update(path string, body []byte) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.data[path] = body
}

// cacheUpdater is a io.ReadCloser wrapping a Respose.Body
// to intercept Read() calls and update cache with downloaded content
type cacheUpdater struct {
	fnUpdate cacheUpdateCallback //  called on Close()
	path     string
	body     io.ReadCloser
	buf      bytes.Buffer
}

type cacheUpdateCallback func(path string, body []byte)

func (c *Cache) NewUpdater(path string, body io.ReadCloser) *cacheUpdater {
	return &cacheUpdater{
		path: path,
		body: body,
		fnUpdate: func(path string, body []byte) {
			c.Update(path, body)
		},
	}
}

func (cu *cacheUpdater) Read(p []byte) (int, error) {
	n, err := cu.body.Read(p)
	if err == nil || err == io.EOF {
		cu.buf.Write(p[:n])
	}
	return n, err
}

// Close() fires the update callback and propagates to body.Close()
func (cu *cacheUpdater) Close() error {
	if cu.body == nil {
		return nil
	}
	cu.fnUpdate(cu.path, cu.buf.Bytes())
	err := cu.body.Close()
	if err != nil {
		return err
	}
	cu.body = nil
	cu.buf.Reset()
	return nil
}
