package mfmap

import (
	"fmt"
	"io"

	"github.com/beevik/etree"

	svt "gometeo/svgtools"
)

// CropRatio is the SVG viewport crop ratio applied to upstream maps.
var CropRatio = svt.CropRatio{
	Left:   0.20,
	Right:  0.08,
	Top:    0.08,
	Bottom: 0.08,
}

func (m *MfMap) ParseSvgMap(r io.Reader) error {
	xml, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("read error: %w", err)
	}
	doc := etree.NewDocument()
	if err = doc.ReadFromBytes(xml); err != nil {
		return fmt.Errorf("xml parse error: %w", err)
	}
	tree := (*svt.Tree)(doc)
	szOrig, err := tree.GetSize()
	if err != nil {
		return fmt.Errorf("could not get svg size: %w", err)
	}
	if err = tree.SetSize(szOrig.Crop(CropRatio)); err != nil {
		return fmt.Errorf("could not set svg size: %w", err)
	}
	buf, err := doc.WriteToBytes()
	if err != nil {
		return fmt.Errorf("xml serialization error: %w", err)
	}
	m.SvgMap = buf
	return nil
}
