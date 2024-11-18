package crawl

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
)

const (
	urlBase       = "https://meteofrance.com"
	sessionCookie = "mfsession"
)

type MfCrawler struct {
	client *MfClient
}

type MfCache map[string][]byte

type MfClient struct {
	auth_token string
	client     *http.Client
	cache      MfCache
}

type MfMap struct {
	nom  string
	html string
}

// Declaring an enum type for cache control
type CachePolicy int

const (
	CacheDefault  CachePolicy = iota // default. return cached data if available, otherwise send a request
	CacheUpdate                      // ignore cache but store response
	CacheDisabled                    // ignore cache and do not store response
	CacheOnly                        // do not send any request, only cache data only
)

// NewClient allocates a *MfClient.
// cache is an optional pre-initialized cache. cache=nil is allowed.
func NewClient(cache MfCache) *MfClient {
	if cache == nil {
		cache = make(MfCache)
	}
	return &MfClient{
		client: &http.Client{},
		cache:  cache,
	}
}

// NewCrawler allocates as *MfCrawler
func NewCrawler() *MfCrawler {
	return &MfCrawler{
		client: NewClient(nil),
	}
}

// updateAuthToken() extracts and store authentication token from
// a Set-Cookie "mfsession" header present in every response.
// Warns if token changes during a session
func (cl *MfClient) updateAuthToken(resp *http.Response) error {
	var tok string
	for _, cookie := range resp.Cookies() {
		if cookie.Name == sessionCookie {
			tok = cookie.Value
			break
		}
	}
	if tok == "" {
		msg := fmt.Sprintf("Set-Cookie mfsession absent de la réponse. url=%s", resp.Request.URL.String())
		return errors.New(msg)
	}
	if cl.auth_token != "" && cl.auth_token != tok {
		log.Println("Cookie de session modifié.")
	}
	cl.auth_token, _ = Rot13(tok)
	return nil
}

// addUrlBase returns
//   - path (unchanged) if path starts with urlBase
//   - urlBase+path if path starts with a slash
//   - error in other cases
func addUrlBase(path string) (string, error) {
	l := min(len(path), len(urlBase))
	switch {
	case path == "":
		return "", fmt.Errorf("path is empty")
	case path[0:l] == urlBase:
		return path, nil
	case path[0] == '/':
		return urlBase + path, nil
	default:
		return "", fmt.Errorf("path '%s' does not start with '/' or '%s'", path, urlBase)
	}
}

// Get issues a GET request to path, prefixed with 'baseUrl' constant.
// with a basic cache
func (cl *MfClient) Get(path string, cp CachePolicy) ([]byte, error) {

	// cree une requete GET avec l'url et le header d'authentification
	req, err := http.NewRequest("GET", urlBase+path, nil)
	if err != nil {
		msg := fmt.Sprintf("Erreur de création de la requete pour %s", path)
		return nil, errors.New(msg)
	}
	// cherche dans le cache
	if cp == CacheDefault || cp == CacheOnly {
		body, ok := cl.cache[path]
		// renvoie le résultat si présent dans le cache
		if ok {
			return body, nil
		}
	}
	// arrete ici en mode CacheOnly
	if cp == CacheOnly {
		msg := fmt.Sprint("ressource non disponible dans le cache ", path)
		return nil, errors.New(msg)
	}
	// execute la requête
	if cl.auth_token != "" {
		req.Header.Add("Authorization", "Bearer "+cl.auth_token)
	}
	resp, err := cl.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// met à jour le token de session
	err = cl.updateAuthToken(resp)
	if err != nil {
		return nil, err
	}
	// recupere le contenu de la réponse
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// met à jour le cache
	if cp == CacheDefault || cp == CacheUpdate {
		cl.cache[path] = body
	}
	return body, nil
}

// getMap gets a map from remote service or from local cache if available
func (c *MfCrawler) getMap(path string, parent *MfMap) (*MfMap, error) {
	log.Printf("Crawling %s from parent '%s'\n", path, parent.nom)
	resp, err := c.client.Get(path, CacheDefault)
	if err != nil {
		return nil, err
	}
	m := MfMap{html: string(resp)}
	return &m, nil
}

func Run() error {
	crawler := NewCrawler()
	mfmap, err := crawler.getMap("/", &MfMap{nom: "racine"})
	if err != nil {
		return err
	}
	var trunc int = min(int(200), len(mfmap.html))
	fmt.Printf(mfmap.html[0:trunc])
	return nil
}
