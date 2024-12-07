package mfmap

import (
	"bytes"
	"io"
	"os"
	"reflect"
	"testing"
	"text/template"

	"github.com/beevik/etree"
)

const fileSvgRacine = "pays007.svg"

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

func testCropSVG(t *testing.T, file string) (*svgTree, *bytes.Buffer) {
	name := assets_path + file
	f, err := os.Open(name)
	if err != nil {
		t.Fatalf("os.Open(%s) error: %s", name, err)
	}
	defer f.Close()

	buf := bytes.Buffer{}
	tree, err := readSVG(io.TeeReader(f, &buf))
	if err != nil {
		t.Fatalf("parse error on '%s': %s", name, err)
	}
	return tree, &buf
}

func TestCropSVG(t *testing.T) {
	file := fileSvgRacine
	tree, buf := testCropSVG(t, file)
	sz_orig, err := tree.getSize()
	if err != nil {
		t.Fatalf("cant get original '%s' size: %s", file, err)
	}
	want := sz_orig.crop()
	// call cropSVG() to be tested
	cropReader, err := cropSVG(buf)
	if err != nil {
		t.Fatalf("error while cropping %s: %s", file, err)
	}
	// parse result to get size
	tree, err = readSVG(cropReader)
	if err != nil {
		t.Fatalf("parse error on cropped '%s': %s", file, err)
	}
	got, err := tree.getSize()
	if err != nil {
		t.Fatalf("error while getting cropped size: %s", err)
	}
	// check results
	if !reflect.DeepEqual(*got, want) {
		t.Fatalf("got:%v want:%v", *got, want)
	}
	// bonus : write cropped file to check with a browser
	if err := (*etree.Document)(tree).WriteToFile(assets_path + "cropped.svg"); err != nil {
		t.Error(err)
	}
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

/*
TODO: déplacer ici ce code venu de geography_test.go

func TestSvgQuery(t *testing.T) {
t.Run("svgURL", func(t *testing.T) {
	u, err := m.SvgURL()
	if err != nil {
		t.Fatalf("svgURL() error: %s", err)
	}
	expr := regexp.MustCompile(svgRegexp)
	if !expr.Match([]byte(u.String())) {
		t.Errorf("svgUrl()='%s' does not match '%s'", u.String(), svgRegexp)
	}
})
}

*/
