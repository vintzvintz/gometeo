package mfmap

import (
	"fmt"
	"regexp"
	"testing"
)

func TestParseGeography(t *testing.T) {
	file := "geography.json"
	j := openFile(t, file)
	defer j.Close()

	geo, err := parseGeography(j)

	if err != nil {
		t.Fatal(fmt.Errorf("parseGeography() error: %w", err))
	}
	if geo == nil {
		t.Fatal("parseGeography() returned no data")
	}
}

const (
	geoRegexp = `^https://meteofrance.com/modules/custom/mf_map_layers_v2/maps/desktop/[A-Z]+/geo_json/[a-z0-9]+-aggrege.json$`
	svgRegexp = `^https://meteofrance.com/modules/custom/mf_map_layers_v2/maps/desktop/[A-Z]+/[a-z0-9]+.svg`
)

func TestGeographyQuery(t *testing.T) {

	m := parseHtml(t, fileHtmlRacine)

	t.Run("geographyURL", func(t *testing.T) {
		u, err := m.geographyURL()
		if err != nil {
			t.Fatalf("geographyURL() error: %s", err)
		}
		expr := regexp.MustCompile(geoRegexp)
		if !expr.Match([]byte(u.String())) {
			t.Errorf("geographyUrl()='%s' does not match '%s'", u.String(), geoRegexp)
		}
	})

	t.Run("svgURL", func(t *testing.T) {
		u, err := m.svgURL()
		if err != nil {
			t.Fatalf("svgURL() error: %s", err)
		}
		expr := regexp.MustCompile(svgRegexp)
		if !expr.Match([]byte(u.String())) {
			t.Errorf("svgUrl()='%s' does not match '%s'", u.String(), svgRegexp)
		}
	})
}

//const svgTestFile = "pays007.svg"

func TestPxToInt(t *testing.T) {

	type testType struct {
		px       string
		want_n   int
		want_err bool
	}

	tests := []testType{
		{"", 0, true},      // error
		{"wesh", 0, true},  // error
		{"px", 0, true},    // error
		{"51", 0, true},    // error
		{"51 px", 0, true}, // error
		{" 51px", 0, true}, // error
		{"51px ", 0, true}, // error
		{"51px", 51, false},
		{"0px", 0, false},
		{"1px", 1, false},
		{"-0px", 0, false},
		{"-1px", -1, false},
	}
	for _, test := range tests {
		t.Run(test.px, func(t *testing.T) {
			got_n, err := pxToInt([]byte(test.px))
			if (err != nil) != test.want_err {
				t.Fatalf("pxToInt(%s) (error!=nil)='%s' want '%v'", test.px, err, test.want_err)
			}
			if got_n != test.want_n {
				t.Fatalf("pxToInt(%s)=%d want %d", test.px, got_n, test.want_n)
			}
		})
	}
}

/*
func regexpIntegers(t *testing.T, b []byte, expr string) []int {
	re := regexp.MustCompile(expr)
	matches := re.FindSubmatch(b)
	if len(matches) < 2 {
		t.Fatalf("regexp '%s' has no capturing group or it did not match on: '%s'", re.String(), string(b))
	}
	vals := make([]int, len(matches)-1)
	for i, txt := range matches[1:] {
		n, err := strconv.Atoi(string(txt))
		if err != nil {
			t.Error(err)
		}
		vals[i] = n
	}
	return vals
}
*/
/*
func regexpGetSize(t *testing.T, svg []byte) svgSize {
	width := regexpIntegers(t, svg, `width="(\d+)px"`)
	height := regexpIntegers(t, svg, `height="(\d+)px"`)
	viewbox := regexpIntegers(t, svg, `viewBox="(\d+)\s+(\d+)\s+(\d+)\s+(\d+)"`)
	return svgSize{
		width:   width[0],
		height:  height[0],
		viewbox: [4]int(viewbox[0:]),
	}
}
*/
/*
func TestCropSVG(t *testing.T) {
	name := assets_path + svgTestFile

	f, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("could not read %s: %s", name, err)
	}

	s := regexpGetSize(t, f)
	cropped, err := cropSVG(bytes.NewReader(f))
	if err != nil {
		t.Fatalf("could not crop %s: %s", name, err)
	}
	svg, _ := io.ReadAll(cropped)
	t.Log(string(svg[:400]))

	got := regexpGetSize(t, svg)

	X, Y := float64(s.viewbox[2]), float64(s.viewbox[3])
	vb := [4]int{
		s.viewbox[0] + int(X*crop_left),
		s.viewbox[1] + int(Y*crop_top),
		s.viewbox[2] - int(X*(crop_right+crop_left)),
		s.viewbox[3] - int(Y*(crop_top+crop_bottom)),
	}

	want := svgSize{
		height:  vb[2],
		width:   vb[3],
		viewbox: vb,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("svgCrop(%v) has size %v want %v", s, got, want)
	}
}
*/
