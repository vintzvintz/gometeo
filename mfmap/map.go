package mfmap

import (
	"fmt"
	"io"
)

type MfMap struct {
	nom  string
	parent *MfMap
	html string
}

// accessor
func (m *MfMap) Nom() string {
	return m.nom
}

// accessor
func (m *MfMap) Html() string {
	return m.html
}

// accessor
func (m *MfMap) SetParent( parent *MfMap ) {
	m.parent = parent
}

func NewFrom(io.Reader) (*MfMap, error) {
	return nil, fmt.Errorf("fail")
}
