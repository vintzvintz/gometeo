package mfmap

import "log"


func (m *MfMap)Merge(new *MfMap) *MfMap {
	log.Printf("Merge() m=%s new=%s", m.Path(), new.Path())
	return new

}