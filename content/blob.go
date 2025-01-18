package content

import (
	"encoding/gob"
	"gometeo/mfmap"
	"log"
	"os"
)

// utility type to store a MeteoContent without the ServeMux and exposed fields
type meteoBlob struct {
	Maps   []*mfmap.MfMap
	Pictos []Picto
}

// LoadBlob ans SaveBlob are useful for dev and maintenance
// panics is any error happens
func LoadBlob(fname string) *Meteo {
	// load and decode the whole blob
	f, err := os.Open(fname)
	if err != nil {
		log.Println(err)
		return nil
	}
	dec := gob.NewDecoder(f)
	blob := meteoBlob{}
	err = dec.Decode(&blob)
	if err != nil {
		log.Println(err)
		return nil
	}
	mc := New()
	for _, m := range blob.Maps {
		mc.maps.update(m)
	}
	for _, p := range blob.Pictos {
		mc.pictos.update(p)
	}
	mc.rebuildMux()
	log.Printf("loaded blob from %s", fname)
	return mc
}

// SaveBlob and LoadBlob are useful for dev and maintenance
// panics is any error happens
func (mc *Meteo) SaveBlob(fname string) {

	f, err := os.Create(fname)
	if err != nil {
		panic(err)
	}
	enc := gob.NewEncoder(f)

	blob := meteoBlob{
		Maps:   mc.maps.asSlice(),
		Pictos: mc.pictos.asSlice(),
	}
	err = enc.Encode(blob)
	if err != nil {
		panic(err)
	}
	log.Printf("blob stored to %s", fname)
}

// for load/save as binary blob
func (ps *pictoStore) asSlice() []Picto {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	pictos := make([]Picto, len(ps.store))
	for name, img := range ps.store {
		pictos = append(pictos, Picto{Name: name, Img: img})
	}
	return pictos
}

// for load/save as binary blob
func (ms *mapStore) asSlice() []*mfmap.MfMap {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	maps := make([]*mfmap.MfMap, 0, len(ms.store))
	for _, m := range ms.store {
		maps = append(maps, m)
	}
	return maps
}
