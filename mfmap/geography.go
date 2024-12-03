package mfmap

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"

	"github.com/beevik/etree"
)

type geoCollection struct {
	Type     FeatureCollectionType `json:"type"`
	Bbox     *Bbox                 `json:"bbox"`
	Features []*geoFeature         `json:"features"`
}

type Bbox struct {
	A, B Coordinates
}

type geoFeature struct {
	Bbox       *Bbox       `json:"bbox"`
	Type       FeatureType `json:"type"`
	Properties GeoProperty `json:"properties"`
	Geometry   GeoGeometry `json:"geometry"`
}

type GeoProperty struct {
	Prop0 Prop0 `json:"prop0"`
	// Prop1 Prop1 `json:"prop1"`
	// Prop2 Prop2 `json:"prop2"`
}

type GeoGeometry struct {
	Type   PolygonType     `json:"type"`
	Coords [][]Coordinates `json:"coordinates"`
}

type Prop0 struct {
	Nom   string `json:"nom"`
	Cible string `json:"cible"`
	Paths Paths  `json:"paths"`
}

type Paths struct {
	Fr string `json:"fr"`
	En string `json:"en"`
	Es string `json:"es"`
}

type PolygonType string

const polygonStr = "Polygon"

/*
const (
	crop_left   = 0.20
	crop_right  = 0.08
	crop_top    = 0.08
	crop_bottom = 0.08
)
*/

type svgSize struct {
	Height  int
	Width   int
	Viewbox vbType
}

type vbType [4]int

func (vb vbType) String() string {
	return fmt.Sprintf("%d %d %d %d", vb[0], vb[1], vb[2], vb[3])
}

/*
	var cropParams = struct {
		left, bottom, right, top float64
	}{0.20, 0.08, 0.08, 0.08}
*/
func (bbox *Bbox) UnmarshalJSON(b []byte) error {
	var a [4]float64
	if err := json.Unmarshal(b, &a); err != nil {
		return fmt.Errorf("bbox unmarshal error: %w. Want a [4]float64 array", err)
	}
	p1, err := NewCoordinates(a[0], a[1])
	if err != nil {
		return err
	}
	p2, err := NewCoordinates(a[2], a[3])
	if err != nil {
		return err
	}
	bbox.A, bbox.B = *p1, *p2
	return nil
}

func (pt *PolygonType) UnmarshalJSON(b []byte) error {
	s, err := unmarshalConstantString(b, polygonStr, "GeoGeometry.Type")
	if err != nil {
		return err
	}
	*pt = PolygonType(s)
	return nil
}

func parseGeography(r io.Reader) (*geoCollection, error) {
	var gc geoCollection
	j, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("could not read geography data: %w", err)
	}
	err = json.Unmarshal(j, &gc)
	if err != nil {
		return nil, fmt.Errorf("invalid geography: %w", err)
	}
	return &gc, nil
}

func pxToInt(px []byte) (int, error) {
	const expr = `^([0-9-]+)px$` // width and height are "666px"-like
	re := regexp.MustCompile(expr)
	m := re.FindSubmatch(px)
	if m == nil {
		return 0, fmt.Errorf("'%s' does not match `%s`", string(px), expr)
	}
	n, err := strconv.Atoi(string(m[1])) // m[0] is the full match
	if err != nil {
		return 0, err
	}
	return n, nil
}

func viewboxToInt(b []byte) (vbType, error) {
	const expr = `^([0-9]+) ([0-9]+) ([0-9]+) ([0-9]+)$` // 4 integers expected
	re := regexp.MustCompile(expr)
	m := re.FindSubmatch(b)
	if m == nil {
		return vbType{}, fmt.Errorf("'%s' does not match `%s`", string(b), expr)
	}
	var vb vbType
	for i := 0; i < 4; i++ {
		n, err := strconv.Atoi(string(m[i+1])) // m[0] is the full match
		if err != nil {
			return vbType{}, fmt.Errorf("cant parse '%s' into [4]int : %w", string(m[i+1]), err)
		}
		vb[i] = n
	}
	return vb, nil
}

func getSvgSize(doc *etree.Document) (*svgSize, error) {

	// get root element
	if doc == nil {
		return nil, fmt.Errorf("null pointer")
	}
	root := doc.SelectElement("svg")
	if root == nil {
		return nil, fmt.Errorf("<svg> root element not found")
	}

	// extract width & height
	attrToInt := func(a string) (int, error) {
		attr := root.SelectAttr(a)
		if attr == nil {
			return 0, fmt.Errorf("attr %s not found", a)
		}
		return pxToInt([]byte(attr.Value))
	}
	h, err := attrToInt("height")
	if err != nil {
		return nil, err
	}
	w, err := attrToInt("width")
	if err != nil {
		return nil, err
	}
	// extract viewbox
	attr_vb := root.SelectAttr("viewBox")
	if attr_vb == nil {
		return nil, fmt.Errorf("viewBox attr not found")
	}
	vb, err := viewboxToInt([]byte(attr_vb.Value))
	if err != nil {
		return nil, err
	}

	return &svgSize{Height: h, Width: w, Viewbox: vb}, nil
}

/*
func cropSVG(svg io.Reader) (io.Reader, error) {
	xml, err := io.ReadAll(svg)
	if err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}
	doc := etree.NewDocument()
	err = doc.ReadFromBytes(xml)
	if err != nil {
		return nil, fmt.Errorf("xml parse error: %w", err)
	}

		s, err := getSvgSize(doc)
		if err != nil {
			return nil, fmt.Errorf("could not get svg size: %w", err)
		}
		_ = s

	cropped, err := doc.WriteToBytes()
	if err != nil {
		return nil, fmt.Errorf("xml write error: %w", err)
	}
	return bytes.NewReader(cropped), nil
}
*/
/*
   vb_crop = [ viewbox[0]+crop_O*viewbox[2],  \
               viewbox[1]+crop_N*viewbox[3],  \
               viewbox[2]-(crop_O+crop_E)*viewbox[2],   \
               viewbox[3]-(crop_N+crop_S)*viewbox[3] ]
*/
