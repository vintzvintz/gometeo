package mfmap

import (
	"io"
	"strings"
)

type MfMap struct {
	nom    string
	parent *MfMap
	html   []byte
}

// accessor
func (m *MfMap) Nom() string {
	return m.nom
}

/*
// accessor
func (m *MfMap) Html() string {
	return m.html
}
*/

// accessor
func (m *MfMap) SetParent(parent *MfMap) {
	m.parent = parent
}

func NewFrom(r io.Reader) (*MfMap, error) {
	//return nil, fmt.Errorf("fail")
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return &MfMap{html: buf}, nil
}

func (m *MfMap) JsonContent() (io.Reader, error) {

	// tokenizer
	
	return strings.NewReader("{le_vincent:1}"), nil
}