package svgtools

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/beevik/etree"
)

type Tree etree.Document

type Root etree.Element

type Size struct {
	Width   int
	Height  int
	Viewbox Viewbox
}

type Viewbox [4]int

type CropRatio struct {
	Left, Right, Top, Bottom float64
}

const (
	xmlRoot    = "svg"
	xmlHeight  = "height"
	xmlWidth   = "width"
	xmlViewbox = "viewBox"
)

// String() serialises a vbType into an XML attribute value
func (vb Viewbox) String() string {
	return fmt.Sprintf("%d %d %d %d", vb[0], vb[1], vb[2], vb[3])
}

func (sz Size) Crop(cr CropRatio) Size {
	newW := int(float64(sz.Viewbox[2]) * (1 - cr.Left - cr.Right))
	newH := int(float64(sz.Viewbox[3]) * (1 - cr.Top - cr.Bottom))

	return Size{
		Width:  newW,
		Height: newH,
		Viewbox: Viewbox{
			sz.Viewbox[0] + int(float64(sz.Viewbox[2])*cr.Left),
			sz.Viewbox[1] + int(float64(sz.Viewbox[3])*cr.Top),
			newW,
			newH,
		},
	}
}

func (doc *Tree) GetSize() (sz Size, err error) {
	// get root element
	root, err := doc.getRoot()
	if err != nil {
		return
	}
	// extract width & height & viewBox
	h, err := root.getHeight()
	if err != nil {
		return
	}
	w, err := root.getWidth()
	if err != nil {
		return
	}
	vb, err := root.getViewbox()
	if err != nil {
		return
	}
	return Size{Height: h, Width: w, Viewbox: vb}, nil
}

func (doc *Tree) SetSize(sz Size) error {
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

// pxToInt
func parsePx(px []byte) (int, error) {
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

// ViewboxToInt
func parseViewbox(b []byte) (Viewbox, error) {
	const expr = `^([0-9]+) ([0-9]+) ([0-9]+) ([0-9]+)$` // 4 integers expected
	re := regexp.MustCompile(expr)
	m := re.FindSubmatch(b)
	if m == nil {
		return Viewbox{}, fmt.Errorf("'%s' does not match `%s`", string(b), expr)
	}
	var vb Viewbox
	for i := 0; i < 4; i++ {
		n, err := strconv.Atoi(string(m[i+1])) // m[0] is the full match
		if err != nil {
			return Viewbox{}, fmt.Errorf("cant parse '%s' into [4]int : %w", string(m[i+1]), err)
		}
		vb[i] = n
	}
	return vb, nil
}

func (doc *Tree) getRoot() (*Root, error) {
	root := doc.SelectElement(xmlRoot)
	if root == nil {
		return nil, fmt.Errorf("<svg> root element not found")
	}
	return (*Root)(root), nil
}

func (root *Root) getAttr(a string) (*etree.Attr, error) {
	attr := (*etree.Element)(root).SelectAttr(a)
	if attr == nil {
		return nil, fmt.Errorf("attr %s not found", a)
	}
	return attr, nil
}

func (root *Root) setAttr(name, val string) error {
	attr, err := root.getAttr(name)
	if err != nil {
		return err
	}
	attr.Value = val
	return nil
}

func (root *Root) getHeight() (int, error) {
	attr, err := root.getAttr(xmlHeight)
	if err != nil {
		return 0, err
	}
	return parsePx([]byte(attr.Value))
}

func (root *Root) setHeight(h int) error {
	return root.setAttr(xmlHeight, fmt.Sprintf("%dpx", h))
}

func (root *Root) getWidth() (int, error) {
	attr, err := root.getAttr(xmlWidth)
	if err != nil {
		return 0, err
	}
	return parsePx([]byte(attr.Value))
}

func (root *Root) setWidth(w int) error {
	return root.setAttr(xmlWidth, fmt.Sprintf("%dpx", w))
}

func (root *Root) getViewbox() (Viewbox, error) {
	attr, err := root.getAttr(xmlViewbox)
	if err != nil {
		return Viewbox{}, err
	}
	return parseViewbox([]byte(attr.Value))
}

func (root *Root) setViewbox(vb Viewbox) error {
	return root.setAttr(xmlViewbox, vb.String())
}
