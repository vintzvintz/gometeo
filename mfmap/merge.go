package mfmap

import "log"


func Merge(old, new *MfMap) *MfMap {
	if old == nil {
		return new
	}
	log.Printf("Merge() old=%s new=%s", old.Path(), new.Path())
	return new
}