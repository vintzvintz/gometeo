package crawl

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	baseUrl         string
	noSessionCookie bool // do not expect mfsession cookie
	authToken       string
	client          *http.Client
	cache           *Cache
}

const userAgentFirefox = "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:133.0) Gecko/20100101 Firefox/133.0"

// NewClient allocates a *MfClient.
// cache is an optional pre-initialized cache.
func NewClient(baseUrl string) *Client {
	return &Client{
		baseUrl: baseUrl,
		client:  &http.Client{},
		cache:   &Cache{},
	}
}

// custom error for MfClient
type MissingCookieError string

func (e MissingCookieError) Error() string {
	return "MissingCookieError: " + string(e)
}

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
	closed bool
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
	if cu.closed {
		return nil
	}
	cu.closed = true
	cu.cache[cu.path] = cu.buf
	cu.buf = nil
	cu.body.Close()
	cu.body = nil
	return nil
}

// updateAuthToken() extracts and store authentication token from
// a Set-Cookie "mfsession" header present in every response.
// Warns if token changes during a session
func (cl *Client) updateAuthToken(resp *http.Response) error {
	if cl.noSessionCookie { // do not expect a session cookie from api server (rpcache-aa)
		return nil
	}
	var tok string
	for _, cookie := range resp.Cookies() {
		if cookie.Name == sessionCookie {
			tok = cookie.Value
			break
		}
	}
	if tok == "" {
		msg := fmt.Sprintf("no Set-Cookie mfsession from url '%s'", resp.Request.URL.String())
		return MissingCookieError(msg)
	}
	tok, _ = Rot13(tok)
	/*if cl.authToken != "" && cl.authToken != tok {
		log.Println("Cookie de session modifié.")
	}*/
	cl.authToken = tok
	return nil
}

// addUrlBase returns
//   - path unchanged if path starts with base
//   - base+path if path starts with a slash
//   - error in other cases
func (cl *Client) addUrlBase(path string) (string, error) {
	base := cl.baseUrl
	l := min(len(path), len(base))
	switch {
	case len(base) == 0:
		return "", fmt.Errorf("invalid or empty baseUrl: '%s'", base)
	case path == "":
		return "", fmt.Errorf("empty path")
	case path[0:l] == base:
		return path, nil
	case path[0] == '/':
		return base + path, nil
	default:
		return "", fmt.Errorf("path '%s' invalid, must start with '/' or '%s'", path, base)
	}
}

// Get issues a GET request to path, prefixed with 'baseUrl' constant.
// implement a basic cache, controlled with policy parameter
func (cl *Client) Get(path string, policy CachePolicy) (io.ReadCloser, error) {
	// commence par chercher dans le cache avant de lancer la requete
	// le cache est ignoré avec CacheDisabled et CacheUpdate
	if policy == CacheDefault || policy == CacheOnly {
		body, ok := cl.cache.lookup(path)
		if ok {
			return body, nil
		}
	}
	// arrete ici en mode CacheOnly
	if policy == CacheOnly {
		msg := fmt.Sprint("ressource non disponible dans le cache ", path)
		return nil, errors.New(msg)
	}
	// cree une requete GET sur path
	url, err := cl.addUrlBase(path)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		msg := fmt.Sprintf("erreur de création de la requête http pour %s", path)
		return nil, errors.New(msg)
	}
	// execute la requête avec le token d'authentification et un user agent courant
	if cl.authToken != "" {
		req.Header.Add("Authorization", "Bearer "+cl.authToken)
	}
	req.Header.Add("user-agent", userAgentFirefox)

	resp, err := cl.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s on '%s'", resp.Status, resp.Request.URL)
	}
	//log.Printf("request '%s' %d", resp.Request.URL, resp.StatusCode)
	// met à jour le token de session
	err = cl.updateAuthToken(resp)
	if err != nil {
		return nil, err
	}
	// met à jour le cache
	if policy == CacheDefault || policy == CacheUpdate {
		return cl.cache.NewUpdater(path, resp.Body), nil
	}
	return resp.Body, nil
}
