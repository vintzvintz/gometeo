package mfmap

import (
	"bytes"
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

const (
	cropLeftPx   = 144
	cropRightPx  = 58
	cropTopPx    = 45
	cropBottomPx = 45
)

type svgSize struct {
	Width   int
	Height  int
	Viewbox vbType
}

type vbType [4]int

func (vb vbType) String() string {
	return fmt.Sprintf("%d %d %d %d", vb[0], vb[1], vb[2], vb[3])
}

func (sz svgSize) crop() svgSize {
	w := sz.Width - cropLeftPx - cropRightPx
	h := sz.Height - cropTopPx - cropBottomPx
	return svgSize{
		Width:   w,
		Height:  h,
		Viewbox: vbType{cropLeftPx, cropTopPx, w, h},
	}
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

type svgTree etree.Document
type svgRoot etree.Element

const (
	xmlRoot    = "svg"
	xmlHeight  = "height"
	xmlWidth   = "width"
	xmlViewbox = "viewBox"
)

func (doc *svgTree) getRoot() (*svgRoot, error) {
	root := doc.SelectElement(xmlRoot)
	if root == nil {
		return nil, fmt.Errorf("<svg> root element not found")
	}
	return (*svgRoot)(root), nil
}

func (root *svgRoot) getAttr(a string) (*etree.Attr, error) {
	attr := (*etree.Element)(root).SelectAttr(a)
	if attr == nil {
		return nil, fmt.Errorf("attr %s not found", a)
	}
	return attr, nil
}

func (root *svgRoot) setAttr(name, val string) error {
	attr := (*etree.Element)(root).SelectAttr(name)
	if attr == nil {
		return fmt.Errorf("attr %s not found", name)
	}
	attr.Value = val
	return nil
}

func (root *svgRoot) getHeight() (int, error) {
	attr, err := root.getAttr(xmlHeight)
	if err != nil {
		return 0, err
	}
	return pxToInt([]byte(attr.Value))
}

func (root *svgRoot) setHeight(h int) error {
	return root.setAttr(xmlHeight, fmt.Sprintf("%dpx", h))
}

func (root *svgRoot) getWidth() (int, error) {
	attr, err := root.getAttr(xmlWidth)
	if err != nil {
		return 0, err
	}
	return pxToInt([]byte(attr.Value))
}

func (root *svgRoot) setWidth(w int) error {
	return root.setAttr(xmlWidth, fmt.Sprintf("%dpx", w))
}

func (root *svgRoot) getViewbox() (vbType, error) {
	attr, err := root.getAttr(xmlViewbox)
	if err != nil {
		return vbType{}, err
	}
	return viewboxToInt([]byte(attr.Value))
}

func (root *svgRoot) setViewbox(vb vbType) error {
	return root.setAttr(xmlViewbox, vb.String())
}

func (doc *svgTree) getSize() (*svgSize, error) {
	// get root element
	root, err := doc.getRoot()
	if err != nil {
		return nil, err
	}
	// extract width & height & viewBox
	h, err := root.getHeight()
	if err != nil {
		return nil, err
	}
	w, err := root.getWidth()
	if err != nil {
		return nil, err
	}
	vb, err := root.getViewbox()
	if err != nil {
		return nil, err
	}
	return &svgSize{Height: h, Width: w, Viewbox: vb}, nil
}

func (doc *svgTree) setSize(sz svgSize) error {
	// get root element
	root, err := doc.getRoot()
	if err != nil {
		return err
	}
	// call setter methods
	if err = root.setHeight(sz.Height); err != nil {
		return err
	}
	if err = root.setWidth(sz.Width); err != nil {
		return err
	}
	if err = root.setViewbox(sz.Viewbox); err != nil {
		return err
	}
	return nil
}

func cropSVG(svg io.Reader) (io.Reader, error) {

	xml, err := io.ReadAll(svg)
	if err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}
	// parse original svg to get its original size attributes
	doc := etree.NewDocument()
	err = doc.ReadFromBytes(xml)
	if err != nil {
		return nil, fmt.Errorf("xml parse error: %w", err)
	}
	tree := (*svgTree)(doc)
	szOrig, err := tree.getSize()
	if err != nil {
		return nil, fmt.Errorf("could not get svg size: %w", err)
	}
	// set cropped size attributes to the <svg> root element
	sz := szOrig.crop()
	err = tree.setSize(sz)
	if err != nil {
		return nil, fmt.Errorf("svgTree.setSize(%v) error: %w", sz, err)
	}
	// serialize to a byte slice
	cropped, err := (*etree.Document)(tree).WriteToBytes()
	if err != nil {
		return nil, fmt.Errorf("xml serialization error: %w", err)
	}
	return bytes.NewReader(cropped), nil
}
