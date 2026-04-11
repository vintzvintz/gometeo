package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"gometeo/appconf"
	"gometeo/content"
	"gometeo/mfmap"
	"gometeo/mfmap/schedule"
)

func init() {
	appconf.Init([]string{})
}

const testDataDir = "test_data/"

// TestIntegrationPipeline tests the full path:
// mock upstream → crawler.Fetch() → content.Receive() → HTTP handler responses.
//
// Since the crawler's API client targets a host derived from parsed HTML
// (rpcache-aa.meteofrance.com), and we can't easily redirect HTTPS requests
// to a local mock, we test the pipeline by:
// 1. Building an MfMap from test fixtures (same as the crawler would produce)
// 2. Feeding it through content.Receive() channels
// 3. Verifying the full HTTP handler chain serves correct responses
func TestIntegrationPipeline(t *testing.T) {

	// Build a complete MfMap from test fixtures (equivalent to crawler output)
	m := buildIntegrationMap(t)

	// Create pictos (equivalent to crawler picto output)
	pictos := []mfmap.Picto{
		{Name: "p1j", Img: []byte(`<svg xmlns="http://www.w3.org/2000/svg"><circle r="5"/></svg>`)},
		{Name: "p3j", Img: []byte(`<svg xmlns="http://www.w3.org/2000/svg"><rect width="10" height="10"/></svg>`)},
	}

	// Wire through content.Receive() — the real fan-in pipeline
	dayMin, dayMax := appconf.KeepDays()
	mc := content.New(content.ContentConf{
		DayMin:  dayMin,
		DayMax:  dayMax,
		CacheId: appconf.CacheId(),
	})
	mapCh := make(chan *mfmap.MfMap, 1)
	pictoCh := make(chan mfmap.Picto, len(pictos))

	mapCh <- m
	close(mapCh)
	for _, p := range pictos {
		pictoCh <- p
	}
	close(pictoCh)

	done := mc.Receive(mapCh, pictoCh)
	<-done

	// Verify content store is populated
	if !mc.Ready() {
		t.Fatal("Meteo should be Ready after receiving maps")
	}

	// Start a test HTTP server with the full Meteo handler
	srv := httptest.NewServer(mc)
	defer srv.Close()
	cl := srv.Client()

	// Test 1: HTML page at /france should return 200
	t.Run("html_page", func(t *testing.T) {
		resp, err := cl.Get(srv.URL + "/france")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET /france: status %d, want %d", resp.StatusCode, http.StatusOK)
		}
		ct := resp.Header.Get("Content-Type")
		if !strings.Contains(ct, "text/html") {
			t.Fatalf("Content-Type = %q, want text/html", ct)
		}
		body, _ := io.ReadAll(resp.Body)
		if len(body) == 0 {
			t.Fatal("HTML body is empty")
		}
	})

	// Test 2: JSON data endpoint should return valid JSON
	t.Run("json_data", func(t *testing.T) {
		resp, err := cl.Get(srv.URL + "/france/data")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET /france/data: status %d, want %d", resp.StatusCode, http.StatusOK)
		}
		ct := resp.Header.Get("Content-Type")
		if !strings.Contains(ct, "application/json") {
			t.Fatalf("Content-Type = %q, want application/json", ct)
		}
		body, _ := io.ReadAll(resp.Body)
		// Verify it's valid JSON with expected fields
		var data map[string]interface{}
		if err := json.Unmarshal(body, &data); err != nil {
			t.Fatalf("response is not valid JSON: %v", err)
		}
		// Check key fields exist
		for _, key := range []string{"name", "path", "breadcrumb", "prevs"} {
			if _, ok := data[key]; !ok {
				t.Errorf("JSON response missing key %q", key)
			}
		}
		if data["name"] != "France" {
			t.Errorf("name = %q, want %q", data["name"], "France")
		}
	})

	// Test 3: SVG map endpoint should return SVG
	t.Run("svg_map", func(t *testing.T) {
		svgURL := srv.URL + "/france/" + appconf.CacheId() + "/svg"
		resp, err := cl.Get(svgURL)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET svg: status %d, want %d", resp.StatusCode, http.StatusOK)
		}
		ct := resp.Header.Get("Content-Type")
		if ct != "image/svg+xml" {
			t.Fatalf("Content-Type = %q, want image/svg+xml", ct)
		}
		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "<svg") {
			t.Fatal("SVG response missing <svg tag")
		}
		// SVG should have immutable cache header
		cc := resp.Header.Get("Cache-Control")
		if !strings.Contains(cc, "immutable") {
			t.Fatalf("Cache-Control = %q, want immutable", cc)
		}
	})

	// Test 4: Picto endpoint should serve stored pictos
	t.Run("picto", func(t *testing.T) {
		pictoURL := srv.URL + "/pictos/" + appconf.CacheId() + "/p1j"
		resp, err := cl.Get(pictoURL)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET picto: status %d, want %d", resp.StatusCode, http.StatusOK)
		}
		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "<svg") {
			t.Fatal("picto response missing <svg tag")
		}
	})

	// Test 5: Missing picto should return 404
	t.Run("picto_missing", func(t *testing.T) {
		resp, err := cl.Get(srv.URL + "/pictos/" + appconf.CacheId() + "/nonexistent")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("GET missing picto: status %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
	})

	// Test 6: Root redirect / → /france
	t.Run("root_redirect", func(t *testing.T) {
		// Don't follow redirects
		noRedirectCl := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		resp, err := noRedirectCl.Get(srv.URL + "/")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusMovedPermanently {
			t.Fatalf("GET /: status %d, want %d", resp.StatusCode, http.StatusMovedPermanently)
		}
		loc := resp.Header.Get("Location")
		if loc != "/france" {
			t.Fatalf("Location = %q, want /france", loc)
		}
	})

	// Test 7: Unknown path should return 404
	t.Run("not_found", func(t *testing.T) {
		resp, err := cl.Get(srv.URL + "/nonexistent-region")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("GET /nonexistent-region: status %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
	})

	// Test 8: Status page
	t.Run("status_page", func(t *testing.T) {
		resp, err := cl.Get(srv.URL + "/statusse")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET /statusse: status %d, want %d", resp.StatusCode, http.StatusOK)
		}
	})
}

// buildIntegrationMap creates a fully populated MfMap from test fixtures,
// simulating what the crawler would produce after fetching from upstream.
func buildIntegrationMap(t *testing.T) *mfmap.MfMap {
	t.Helper()

	r := appconf.UpdateRate()
	m := &mfmap.MfMap{
		OriginalPath: "/",
		Conf: mfmap.MapConf{
			CacheId:  appconf.CacheId(),
			VueJs:    appconf.VueJs(),
			Upstream: appconf.Upstream(),
			Rates: schedule.UpdateRates{
				HotDuration: r.HotDuration,
				HotMaxAge:   r.HotMaxAge,
				ColdMaxAge:  r.ColdMaxAge,
			},
		},
	}

	htmlFile, err := os.Open(testDataDir + "racine.html")
	if err != nil {
		t.Fatal(err)
	}
	defer htmlFile.Close()
	if err := m.ParseHtml(htmlFile); err != nil {
		t.Fatal(err)
	}

	geoFile, err := os.Open(testDataDir + "geography.json")
	if err != nil {
		t.Fatal(err)
	}
	defer geoFile.Close()
	if err := m.ParseGeography(geoFile); err != nil {
		t.Fatal(err)
	}

	fcFile, err := os.Open(testDataDir + "multiforecast.json")
	if err != nil {
		t.Fatal(err)
	}
	defer fcFile.Close()
	if err := m.ParseMultiforecast(fcFile); err != nil {
		t.Fatal(err)
	}

	svgFile, err := os.Open(testDataDir + "pays007.svg")
	if err != nil {
		t.Fatal(err)
	}
	defer svgFile.Close()
	if err := m.ParseSvgMap(svgFile); err != nil {
		t.Fatal(err)
	}

	m.Schedule.MarkUpdate()
	return m
}
