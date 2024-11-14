package crawl

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
)

const (
	HTTP_DOMAIN         = "https://meteofrance.com"
	SESSION_COOKIE_NAME = "mfsession"
)

type MfMap struct {
	//nom string
}

type MfClient struct {
	auth_token string
	client     *http.Client
}

type MfCrawler struct {
	client *MfClient
}

// updateAuthToken() extracts and store authentication token from
// a Set-Cookie "mfsession" present in every response.
// Warns if token changes during a session
func (s *MfClient) updateAuthToken(resp *http.Response) error {
	var tok string
	for _, cookie := range resp.Cookies() {
		if cookie.Name == SESSION_COOKIE_NAME {
			tok = cookie.Value
			break
		}
	}
	if tok == "" {
		msg := fmt.Sprintf("Set-Cookie mfsession absent de la réponse. url=%s", resp.Request.URL.String())
		return errors.New(msg)
	}
	if s.auth_token != tok {
		log.Println("Cookie de session modifié.")
	}
	s.auth_token, _ = Rot13(tok)
	return nil
}


// Get issues a GET request to the specified URL
func (s *MfClient) Get(url string) (body []byte, err error) {

	var req *http.Request
	var resp *http.Response

	// initialise un client http pour la premiere requete de la session
	if s.client == nil {
		s.client = &http.Client{}
	}
	// cree une requete GET avec l'url et le header d'authentification
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Erreur de création de la requete pour %s", url)
		return
	}

	// TODO : implémenter un cache ici

	if s.auth_token != "" {
		req.Header.Add("Authorization", "Bearer "+s.auth_token)
	}
	// execute la requête et renvoie le contenu
	resp, err = s.client.Do(req)
	if err == nil {
		body, err = io.ReadAll(resp.Body)
		resp.Body.Close()
	}
	// met à jour le token de session
	if err == nil {
		err = s.updateAuthToken(resp)
	}
	return
}

func (c *MfCrawler) get_map(path string, parent *MfMap) (err error) {

	url := HTTP_DOMAIN + path
	log.Printf("Crawling %s from parent %p\n", url, parent)

	resp, err := c.client.Get(url)
	if err != nil {
		fmt.Printf("Erreur HTTP %s", err)
		return
	}
	log.Print( string(resp[0:200]) )

	return err
}

func NewCrawler( ) *MfCrawler {
	return &MfCrawler{
		client: &MfClient{}, 
	}
}

func Run() {
	crawler := NewCrawler()
	crawler.get_map("/", nil)
}
