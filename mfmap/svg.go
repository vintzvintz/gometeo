package mfmap

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/beevik/etree"

	svt "gometeo/svgtools"
)

var cropPc = svt.CropRatio{
	Left:   0.20,
	Right:  0.08,
	Top:    0.08,
	Bottom: 0.08,
}

// https://meteofrance.com/modules/custom/mf_map_layers_v2/maps/desktop/METROPOLE/pays007.svg
func (m *MfMap) SvgURL() (*url.URL, error) {
	elems := []string{
		"modules",
		"custom",
		"mf_map_layers_v2",
		"maps",
		"desktop",
		m.Data.Info.PathAssets,
		fmt.Sprintf("%s.svg", strings.ToLower(m.Data.Info.IdTechnique)),
	}
	u, err := url.Parse("https://meteofrance.com/" + strings.Join(elems, "/"))
	if err != nil {
		return nil, fmt.Errorf("m.svgURL() error: %w", err)
	}
	return u, nil
}

func (m *MfMap) ParseSvgMap(r io.Reader) error {
	_, buf, err := cropSVG(r)
	if err != nil {
		return err
	}
	m.SvgMap = buf
	return nil
}

// cropSVG and readSVG are separate functions for testing purposes
func ReadSVG(svg io.Reader) (*svt.Tree, []byte, error) {
	xml, err := io.ReadAll(svg)
	if err != nil {
		return nil, nil, fmt.Errorf("read error: %w", err)
	}
	// parse original svg to get its original size attributes
	doc := etree.NewDocument()
	err = doc.ReadFromBytes(xml)
	if err != nil {
		return nil, nil, fmt.Errorf("xml parse error: %w", err)
	}
	return (*svt.Tree)(doc), xml, nil
}

// cropSVG and readSVG are separate functions for testing purposes
func cropSVG(svg io.Reader) (*svt.Tree, []byte, error) {

	// build xml tree from svg
	tree, _, err := ReadSVG(svg)
	if err != nil {
		return nil, nil, err
	}
	szOrig, err := tree.GetSize()
	if err != nil {
		return nil, nil, fmt.Errorf("could not get svg size: %w", err)
	}
	// set cropped size attributes to the <svg> root element
	sz := szOrig.Crop(cropPc)
	err = tree.SetSize(sz)
	if err != nil {
		return nil, nil, fmt.Errorf("svgTree.setSize(%v) error: %w", sz, err)
	}
	// serialize to a byte slice
	cropped, err := (*etree.Document)(tree).WriteToBytes()
	if err != nil {
		return nil, nil, fmt.Errorf("xml serialization error: %w", err)
	}
	return tree, cropped, nil
}
