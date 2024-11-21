package mfmap

import (
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

/*
	func NewMap() *MfMap {
		return &MfMap{}
	}
*/
func (m *MfMap) ReadFrom(io.Reader) (int64, error) {
	return 0, nil
}
