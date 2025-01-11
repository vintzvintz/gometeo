package mfmap

import (
	"bytes"
	"gometeo/testutils"
	"io"
	"reflect"
	"testing"
	"text/template"

	"github.com/beevik/etree"
)

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

func TestViewboxToInt(t *testing.T) {
	type testType struct {
		s        string
		want_vb  vbType
		want_err bool
	}
	zero := vbType{}
	tests := []testType{
		{"", zero, true},         // error
		{"wesh", zero, true},     // error
		{"0", zero, true},        // error
		{"0000", zero, true},     // error
		{"51515151", zero, true}, // error
		{"0 0 0 0 ", zero, true}, // error
		{" 0 0 0 0", zero, true}, // error
		{"0 0 0 0", zero, false},
		{"51 51 51 51", vbType{51, 51, 51, 51}, false},
	}
	for _, test := range tests {
		t.Run(test.s, func(t *testing.T) {
			got_vb, err := viewboxToInt([]byte(test.s))
			if (err != nil) != test.want_err {
				t.Fatalf("viewboxToInt(%s) (error!=nil)='%s' want '%v'", test.s, err, test.want_err)
			}
			if !reflect.DeepEqual(got_vb, test.want_vb) {
				t.Fatalf("viewboxToInt(%s)=%v want %v", test.s, got_vb, test.want_vb)
			}
		})
	}

}
func TestGetSvgSize(t *testing.T) {
	var svgTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd">
<svg version="1.1" xmlns:xlink="http://www.w3.org/1999/xlink" x="0px" y="0px" width="{{.Width}}px" height="{{.Height}}px" viewBox="{{.Viewbox}}" preserveAspectRatio="none" fill="none" xmlns="http://www.w3.org/2000/svg">
</svg>`

	var bullshitTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd">
<svg>bullshit</svg>`

	var svgTests = map[string]struct {
		tdef      string
		want_size *svgSize
		want_err  bool
	}{
		"nil":      {"", nil, true},                      // error
		"empty":    {"", &svgSize{}, true},               // error
		"bullshit": {bullshitTemplate, &svgSize{}, true}, // error
		"zeroes":   {svgTemplate, &svgSize{}, false},
		"ones":     {svgTemplate, &svgSize{1, 1, vbType{1, 1, 1, 1}}, false},
		"rect":     {svgTemplate, &svgSize{50, 100, vbType{0, 0, 50, 100}}, false},
	}

	for name, test := range svgTests {
		t.Run(name, func(t *testing.T) {
			// build svg from a template
			tmpl := template.Must(template.New(name).Parse(test.tdef))
			buf := &bytes.Buffer{}
			err := tmpl.Execute(buf, test.want_size)
			if err != nil {
				t.Fatal(err)
			}
			b, _ := io.ReadAll(buf)
			// parse xml into etree structure
			doc := etree.NewDocument()
			err = doc.ReadFromBytes(b)
			if err != nil {
				t.Fatal(err)
			}
			// call tested function retrieve to get dimensions from <svg> root element attributes
			tree := (*svgTree)(doc)
			got_size, err := tree.getSize()

			// check restults
			if test.want_err {
				if err == nil {
					t.Fatal("error expected on invalid svg document")
				}
				return // test end here
			}
			if !test.want_err && err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(test.want_size, got_size) {
				t.Fatalf("got %v want %v", got_size, test.want_size)
			}
		})
	}
}

func TestParseSvg(t *testing.T) {

	// use unexported function to get original size
	tree, _, err := readSVG(testutils.SvgReader(t))
	var szOrig *svgSize
	if err == nil {
		szOrig, err = tree.getSize()
	}
	if err != nil {
		t.Fatal(err)
	}
	want := szOrig.crop()

	// call tested code
	cropped := testParseSvg(t)

	//use unexported function to parse the cropped image
	// and get its size
	tree, _, err = readSVG(bytes.NewReader(cropped))
	if err != nil {
		t.Fatalf("error parsing cropped SVG: %s", err)
	}
	got, err := tree.getSize()
	if err != nil {
		t.Fatalf("error while getting cropped size: %s", err)
	}
	// check results
	if !reflect.DeepEqual(*got, want) {
		t.Fatalf("got:%v want:%v", *got, want)
	}
}

// returns a cropped and serialized svg image
func testParseSvg(t *testing.T) []byte {
	//m.ParseSvgMap( testutils.SvgReader(t))

	m := MfMap{}
	err := m.ParseSvgMap(testutils.SvgReader(t))
	if err != nil {
		t.Fatalf("error parsing original SVG: %s", err)
	}
	return m.SvgMap
}

func TestCroppedSize(t *testing.T) {

	w := 724
	h := 565
	szOrig := svgSize{
		Width:   w,
		Height:  h,
		Viewbox: vbType{0, 0, w, h},
	}
	got := szOrig.crop()

	wCrop := int(float64(w) * (1 - cropPcLeft - cropPcRight))
	hCrop := int(float64(h) * (1 - cropPcTop - cropPcBottom))
	want := svgSize{
		Width:  wCrop,
		Height: hCrop,
		Viewbox: vbType{
			int(float64(w) * cropPcLeft),
			int(float64(h) * cropPcTop),
			wCrop,
			hCrop,
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("svgSize.Crop(%v) got%v want %v", szOrig, got, want)
	}
}

func TestSvgURL(t *testing.T) {
	t.Run("map_svg", func(t *testing.T) {
		m := testBuildMap(t)
		name := m.Name()
		u, err := m.SvgURL()
		if err != nil {
			t.Fatal(err)
		}
		got := u.String()
		want := "https://meteofrance.com/modules/custom/mf_map_layers_v2/maps/desktop/METROPOLE/pays007.svg"
		if got != want {
			t.Errorf("svgUrl('%s') got '%s' want '%s'", name, got, want)
		}
	})
}
