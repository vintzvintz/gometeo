package crawl

import (
	"testing"
)

/*
func TestGet(t *testing.T) {

	name := "test_data/racine.html"
	f, err := os.Open(name)
	if err != nil {
		t.Errorf("%s : %v", name, err)
	}

	// setup mock server
	srv := httptest.NewServer( http.HandlerFunc( func ( w http.ResponseWriter, req *http.Request ) {
		_, err := io.Copy(w, f)
		if err != nil {
			t.Error(err)
		}
	} ))
	defer srv.Close()

//	client := NewClient(nil)
//	client.Get( )


}
*/


func TestGetMap(t *testing.T) {

	c := NewCrawler()
	//c.client.cache
	m, _ := c.GetMap( "", nil)
	_ = m



}
