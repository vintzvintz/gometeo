package content

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gometeo/mfmap"
	"gometeo/testutils"
)

var testContentConf = ContentConf{DayMin: -2, DayMax: 2, CacheId: "testcache"}

func TestNew(t *testing.T) {
	mc := New(testContentConf)
	if mc == nil {
		t.Fatal("New(testContentConf) returned nil")
	}
	if mc.maps.store == nil {
		t.Fatal("maps store not initialized")
	}
	if mc.pictos.store == nil {
		t.Fatal("pictos store not initialized")
	}
}

func TestReady(t *testing.T) {
	mc := New(testContentConf)
	if mc.Ready() {
		t.Fatal("empty Meteo should not be Ready")
	}

	// Add a map to make it ready
	m := testutils.BuildTestMap(t)
	mc.maps.update(m, -2, 2)
	if !mc.Ready() {
		t.Fatal("Meteo with one map should be Ready")
	}
}

func TestReceiveMaps(t *testing.T) {
	mc := New(testContentConf)
	ch := make(chan *mfmap.MfMap, 1)
	m := testutils.BuildTestMap(t)
	ch <- m
	close(ch)

	done := mc.ReceiveMaps(ch)
	<-done

	if !mc.Ready() {
		t.Fatal("Meteo should be Ready after receiving a map")
	}
}

func TestReceivePictos(t *testing.T) {
	mc := New(testContentConf)
	ch := make(chan mfmap.Picto, 2)
	ch <- mfmap.Picto{Name: "p1j", Img: []byte("<svg>sun</svg>")}
	ch <- mfmap.Picto{Name: "p2n", Img: []byte("<svg>cloud</svg>")}
	close(ch)

	done := mc.ReceivePictos(ch)
	<-done

	mc.pictos.mutex.Lock()
	defer mc.pictos.mutex.Unlock()
	if len(mc.pictos.store) != 2 {
		t.Fatalf("expected 2 pictos, got %d", len(mc.pictos.store))
	}
}

func TestReceive(t *testing.T) {
	mc := New(testContentConf)
	mapCh := make(chan *mfmap.MfMap, 1)
	pictoCh := make(chan mfmap.Picto, 1)

	m := testutils.BuildTestMap(t)
	mapCh <- m
	close(mapCh)

	pictoCh <- mfmap.Picto{Name: "p1j", Img: []byte("<svg/>")}
	close(pictoCh)

	done := mc.Receive(mapCh, pictoCh)
	<-done

	if !mc.Ready() {
		t.Fatal("Meteo should be Ready after Receive")
	}
	mc.pictos.mutex.Lock()
	pictoCount := len(mc.pictos.store)
	mc.pictos.mutex.Unlock()
	if pictoCount != 1 {
		t.Fatalf("expected 1 picto, got %d", pictoCount)
	}
}

func TestPictoServeHTTP(t *testing.T) {
	mc := New(testContentConf)

	// Add pictos
	mc.pictos.update(mfmap.Picto{Name: "p1j", Img: []byte("<svg>sun</svg>")})
	mc.pictos.update(mfmap.Picto{Name: "p2n", Img: []byte("<svg>cloud</svg>")})

	// Build mux with registered pictos
	mc.rebuildMux()

	srv := httptest.NewServer(mc)
	defer srv.Close()
	cl := srv.Client()

	// existing picto should return 200
	pictoURL := srv.URL + "/pictos/" + testContentConf.CacheId + "/p1j"
	resp, err := cl.Get(pictoURL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: got status %d, want %d", pictoURL, resp.StatusCode, http.StatusOK)
	}
	ct := resp.Header.Get("Content-Type")
	if ct != "image/svg+xml" {
		t.Fatalf("Content-Type = %q, want image/svg+xml", ct)
	}
	cacheCtrl := resp.Header.Get("Cache-Control")
	if !strings.Contains(cacheCtrl, "immutable") {
		t.Fatalf("Cache-Control = %q, want immutable", cacheCtrl)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "<svg>sun</svg>" {
		t.Fatalf("body = %q, want <svg>sun</svg>", string(body))
	}

	// missing picto should return 404
	missingURL := srv.URL + "/pictos/" + testContentConf.CacheId + "/unknown"
	resp2, err := cl.Get(missingURL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusNotFound {
		t.Fatalf("GET %s: got status %d, want %d", missingURL, resp2.StatusCode, http.StatusNotFound)
	}
}

func TestMapStoreUpdateAndBreadcrumbs(t *testing.T) {
	mc := New(testContentConf)

	// Build a test map (this is the "france" root map)
	m := testutils.BuildTestMap(t)
	mc.maps.update(m, -2, 2)

	// Check map is stored
	mc.maps.mutex.Lock()
	storedCount := len(mc.maps.store)
	mc.maps.mutex.Unlock()
	if storedCount != 1 {
		t.Fatalf("expected 1 map in store, got %d", storedCount)
	}

	// Check breadcrumb was built
	mc.maps.mutex.Lock()
	stored := mc.maps.store[m.Path()]
	mc.maps.mutex.Unlock()
	if stored == nil {
		t.Fatalf("map not found at path %q", m.Path())
	}
	if len(stored.Breadcrumb) == 0 {
		t.Fatal("breadcrumb should not be empty after update")
	}
}

func TestUpdatable(t *testing.T) {
	mc := New(testContentConf)

	// Empty store should return empty string
	if got := mc.Updatable(); got != "" {
		t.Fatalf("Updatable() on empty store = %q, want empty", got)
	}

	// Add a map with OriginalPath set (as the crawler would)
	m := testutils.BuildTestMap(t)
	m.OriginalPath = "/previsions-meteo-france/france"
	mc.maps.update(m, -2, 2)

	// Should return the OriginalPath of the map needing update
	got := mc.Updatable()
	if got != m.OriginalPath {
		t.Fatalf("Updatable() = %q, want %q", got, m.OriginalPath)
	}
}

func TestServeHTTP(t *testing.T) {
	mc := New(testContentConf)
	m := testutils.BuildTestMap(t)
	mc.maps.update(m, -2, 2)
	mc.rebuildMux()

	srv := httptest.NewServer(mc)
	defer srv.Close()
	cl := srv.Client()

	// The root map should be accessible at /france
	testutils.CheckStatusCode(t, cl, srv.URL+"/"+m.Path(), http.StatusOK)
}

func TestStatusPage(t *testing.T) {
	mc := New(testContentConf)
	m := testutils.BuildTestMap(t)
	mc.maps.update(m, -2, 2)
	mc.rebuildMux()

	srv := httptest.NewServer(mc)
	defer srv.Close()
	cl := srv.Client()

	resp, err := cl.Get(srv.URL + "/statusse")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /statusse: got status %d, want %d", resp.StatusCode, http.StatusOK)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		t.Fatal("status page body is empty")
	}
}

func TestMapStoreStatus(t *testing.T) {
	mc := New(testContentConf)

	// Empty store
	stats := mc.maps.Status()
	if len(stats) != 0 {
		t.Fatalf("empty store Status() returned %d entries, want 0", len(stats))
	}

	// Add map and check stats
	m := testutils.BuildTestMap(t)
	mc.maps.update(m, -2, 2)
	stats = mc.maps.Status()
	if len(stats) != 1 {
		t.Fatalf("Status() returned %d entries, want 1", len(stats))
	}
	s := stats[0]
	if s.Name == "" {
		t.Fatal("stats Name should not be empty")
	}
	if s.Path == "" {
		t.Fatal("stats Path should not be empty")
	}
}

func TestPictoStoreAsSlice(t *testing.T) {
	mc := New(testContentConf)
	mc.pictos.update(mfmap.Picto{Name: "p1j", Img: []byte("img1")})
	mc.pictos.update(mfmap.Picto{Name: "p2n", Img: []byte("img2")})

	sl := mc.pictos.asSlice()
	if len(sl) != 2 {
		t.Fatalf("asSlice() returned %d pictos, want 2", len(sl))
	}
}

func TestMapStoreAsSlice(t *testing.T) {
	mc := New(testContentConf)
	m := testutils.BuildTestMap(t)
	mc.maps.update(m, -2, 2)

	sl := mc.maps.asSlice()
	if len(sl) != 1 {
		t.Fatalf("asSlice() returned %d maps, want 1", len(sl))
	}
}

func TestMergeOnUpdate(t *testing.T) {
	mc := New(testContentConf)

	// First update
	m1 := testutils.BuildTestMap(t)
	mc.maps.update(m1, -2, 2)

	// Second update with same path should merge
	m2 := testutils.BuildTestMap(t)
	mc.maps.update(m2, -2, 2)

	mc.maps.mutex.Lock()
	storedCount := len(mc.maps.store)
	mc.maps.mutex.Unlock()
	if storedCount != 1 {
		t.Fatalf("after two updates with same path, expected 1 map, got %d", storedCount)
	}
}
