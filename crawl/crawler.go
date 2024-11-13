package crawl

import (
	"fmt"
	"net/http"

)

const (
	HTTP_DOMAIN         = "https://meteofrance.com"
	SESSION_COOKIE_NAME = "mfsession"
)

type MfMap struct {
	nom string
}

type MfSession struct {
	auth_token string
}

type MfCrawler struct {
	session *MfSession
}

func getAuthToken(resp *http.Response) (string, error) {
	for _, cookie := range resp.Cookies() {
		if cookie.Name == SESSION_COOKIE_NAME {
			return cookie.Value, nil
		}
	}
	fmt.Printf("Cookie de session non trouvé")
	return "", nil // TODO erreurs
}

func (c *MfCrawler) get_map(path string, parent *MfMap) (err error) {
	url := HTTP_DOMAIN + path
	fmt.Printf("Crawling %s from parent %p\n", url, parent)

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Erreur HTTP %s", err)
		return
	}
	defer resp.Body.Close()

	obfuscated, err := getAuthToken(resp)
	if err != nil {
		//TODO
	}
	fmt.Printf("obfus_auth = %s\n", obfuscated)
	token, err := Rot13( obfuscated )
	if err != nil {
		return 
	}
	fmt.Printf("rot13_auth = %s\n", token)
	return 
}

func Run() {
	crawler := &MfCrawler{}
	crawler.get_map("/", nil)
}
