package server


import (
	"testing"
	_ "net/http/httptest"
)



func spawnServer(t *testing.T) *MeteoServer {
	t.Helper()
	srv, err := NewMeteoServer()
	if err != nil {
		t.Fatalf("NewMeteoServer() error: %s", err)
	}
	return srv
}


func TestMainPage(t *testing.T){
	_ = spawnServer(t)
}