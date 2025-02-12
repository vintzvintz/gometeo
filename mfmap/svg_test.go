package mfmap_test

import (
	"bytes"
	"gometeo/mfmap"
	svt "gometeo/svgtools"
	"gometeo/testutils"
	"io"
	"reflect"
	"testing"
)

// duplicated from (unexported) mfmap.cropPc
var cropPc = svt.CropRatio{
	Left:   0.20,
	Right:  0.08,
	Top:    0.08,
	Bottom: 0.08,
}

func parseAndGetSize(t *testing.T, r io.ReadCloser) (*svt.Tree, svt.Size) {
	var sz svt.Size
	defer r.Close()
	tree, _, err := mfmap.ReadSVG(r)
	if err == nil {
		sz, err = tree.GetSize()
	}
	if err != nil {
		t.Fatal(err)
	}
	return tree, sz
}

func TestParseSvg(t *testing.T) {
	_, szOrig := parseAndGetSize(t, testutils.SvgReader(t))
	want := szOrig.Crop(cropPc)

	// call tested code
	cropped := io.NopCloser(bytes.NewReader(testParseSvg(t)))
	_, got := parseAndGetSize(t, cropped)
	// check results
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got:%v want:%v", got, want)
	}
}

// returns a cropped and serialized svg image
func testParseSvg(t *testing.T) []byte {
	//m.ParseSvgMap( testutils.SvgReader(t))

	m := mfmap.MfMap{}
	err := m.ParseSvgMap(testutils.SvgReader(t))
	if err != nil {
		t.Fatalf("error parsing original SVG: %s", err)
	}
	return m.SvgMap
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
