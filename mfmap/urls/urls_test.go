package urls_test

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"gometeo/mfmap"
	"gometeo/mfmap/urls"
	"gometeo/testutils"
)

const (
	apiBaseURL     = "https://rwg.meteofrance.com/internet2018client/2.0"
	emptyRegexp    = `^$`
	coordsRegexp   = `^([\d\.],?)+$`
	instantsRegexp = `morning,afternoon,evening,night`
	geoRegexp      = `^https://meteofrance.com/modules/custom/mf_map_layers_v2/maps/desktop/[A-Z]+/geo_json/[a-z0-9]+-aggrege.json$`
	svgRegexp      = `^https://meteofrance.com/modules/custom/mf_map_layers_v2/maps/desktop/[A-Z]+/[a-z0-9]+.svg`
)

const assetsPath = "../../test_data/"

func openFile(t *testing.T, name string) io.ReadCloser {
	t.Helper()
	fp := filepath.Join(assetsPath, name)
	f, err := os.Open(fp)
	if err != nil {
		t.Fatalf("os.Open(%s) failed: %v", fp, err)
	}
	return f
}

func parseHtml(t *testing.T) *mfmap.MapData {
	t.Helper()
	f := openFile(t, "racine.html")
	defer f.Close()
	m := mfmap.MfMap{Conf: testutils.TestConf}
	if err := m.ParseHtml(f); err != nil {
		t.Fatalf("ParseHtml() error: %s", err)
	}
	return m.Data
}

func buildMap(t *testing.T) *mfmap.MfMap {
	t.Helper()
	m := mfmap.MfMap{Conf: testutils.TestConf}
	for _, step := range []struct {
		file string
		fn   func(io.Reader) error
	}{
		{"racine.html", m.ParseHtml},
		{"geography.json", m.ParseGeography},
		{"multiforecast.json", m.ParseMultiforecast},
		{"pays007.svg", m.ParseSvgMap},
	} {
		f := openFile(t, step.file)
		if err := step.fn(f); err != nil {
			f.Close()
			t.Fatalf("parsing %s: %v", step.file, err)
		}
		f.Close()
	}
	return &m
}

func TestApiUrl(t *testing.T) {
	data := parseHtml(t)

	tests := map[string]struct {
		path string
		want string
	}{
		"racine": {
			path: "",
			want: apiBaseURL,
		},
		"slash": {
			path: "/",
			want: apiBaseURL + "/",
		},
		"multiforecast": {
			path: urls.ApiMultiforecast,
			want: apiBaseURL + urls.ApiMultiforecast,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			u, err := urls.ApiUrl(data, test.path, nil)
			if err != nil {
				t.Fatalf("ApiUrl() error: %s", err)
			}
			got := u.String()
			if got != test.want {
				t.Errorf("ApiUrl()='%s' want '%s'", got, test.want)
			}
		})
	}
}

func TestForecastUrl(t *testing.T) {
	data := parseHtml(t)

	validationRegexps := map[string]string{
		"bbox":       emptyRegexp,
		"begin_time": emptyRegexp,
		"end_time":   emptyRegexp,
		"time":       emptyRegexp,
		"instants":   instantsRegexp,
		"liste_id":   coordsRegexp,
	}

	u, err := urls.ForecastUrl(data)
	if err != nil {
		t.Fatalf("ForecastUrl() error: %s", err)
	}
	values := u.Query()
	for key, expr := range validationRegexps {
		re := regexp.MustCompile(expr)
		got, ok := values[key]
		if !ok {
			t.Fatalf("ForecastUrl() query does not have key '%s'", key)
		}
		if len(got) != 1 {
			t.Fatalf("ForecastUrl() query ['%s'] has %d values %q, want 1", key, len(got), got)
		}
		if re.Find([]byte(got[0])) == nil {
			t.Errorf("ForecastUrl() query ['%s']='%s' doesnt match '%s'", key, got[0], expr)
		}
	}
}

func TestGeographyUrl(t *testing.T) {
	m := buildMap(t)

	u, err := urls.GeographyUrl(m.Conf.Upstream, m.Data)
	if err != nil {
		t.Fatalf("GeographyUrl() error: %s", err)
	}
	expr := regexp.MustCompile(geoRegexp)
	if !expr.Match([]byte(u.String())) {
		t.Errorf("GeographyUrl()='%s' does not match '%s'", u.String(), geoRegexp)
	}
}

func TestSvgUrl(t *testing.T) {
	m := buildMap(t)

	u, err := urls.SvgUrl(m.Conf.Upstream, m.Data)
	if err != nil {
		t.Errorf("SvgUrl() error: %s", err)
		return
	}
	expr := regexp.MustCompile(svgRegexp)
	if !expr.Match([]byte(u.String())) {
		t.Errorf("SvgUrl()='%s' does not match '%s'", u.String(), svgRegexp)
	}
}
