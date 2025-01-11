package svgtools_test

import (
	"bytes"
	"html/template"
	"io"
	"reflect"
	"testing"

	"github.com/beevik/etree"

	svt "gometeo/svgtools"
)


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
		want_size svt.Size
		want_err  bool
	}{
//		"nil":      {"", nil, true},                      // error
		"empty":    {"", svt.Size{}, true},               // error
		"bullshit": {bullshitTemplate, svt.Size{}, true}, // error
		"zeroes":   {svgTemplate, svt.Size{}, false},
		"ones":     {svgTemplate, svt.Size{1, 1, svt.Viewbox{1, 1, 1, 1}}, false},
		"rect":     {svgTemplate, svt.Size{50, 100, svt.Viewbox{0, 0, 50, 100}}, false},
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
			tree := (*svt.Tree)(doc)
			got_size, err := tree.GetSize()

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

func TestCrop(t *testing.T) {
	w := 724
	h := 565
	cr := svt.CropRatio{   // random percentages
		Left: 0.15,
		Right: 0.25,
		Top: 0.075,
		Bottom: 0.2,
	}
	szOrig := svt.Size{
		Width:   w,
		Height:  h,
		Viewbox: svt.Viewbox{0, 0, w, h},
	}
	got := szOrig.Crop(cr)
	wCrop := int(float64(w) * (1 - cr.Left - cr.Right))
	hCrop := int(float64(h) * (1 - cr.Top - cr.Bottom))
	want := svt.Size{
		Width:  wCrop,
		Height: hCrop,
		Viewbox: svt.Viewbox{
			int(float64(w) * cr.Left),
			int(float64(h) * cr.Top),
			wCrop,
			hCrop,
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("svgSize.Crop(%v) got%v want %v", szOrig, got, want)
	}
}
