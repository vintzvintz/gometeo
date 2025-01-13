package crawl

import (
	"bytes"
	"encoding/gob"
	"gometeo/mfmap"
	"log"
	"os"
)

// utility type to store a MeteoContent without the ServeMux and exposed fields
type meteoBlob struct {
	Maps   []byte
	Pictos []byte
}

func LoadBlob(fname string) *MeteoContent {
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

	// decode maps
	var maps map[string]*mfmap.MfMap
	dec = gob.NewDecoder(bytes.NewReader(blob.Maps))
	err = dec.Decode(&maps)
	if err != nil {
		log.Println(err)
		return nil
	}

	// decode pictos
	var pictos map[string][]byte
	dec = gob.NewDecoder(bytes.NewReader(blob.Pictos))
	err = dec.Decode(&pictos)
	if err != nil {
		log.Println(err)
		return nil
	}

	mc := newContent() // empty but non-nil
	mc.Import(maps, pictos)

	log.Printf("loaded maps & pictos from %s", fname)
	return mc
}

func (mc *MeteoContent) SaveBlob(fname string) {

	blob := meteoBlob{
		Maps:   mc.maps.ToBlob(),
		Pictos: mc.pictos.ToBlob(),
	}

	f, err := os.Create(fname)
	if err != nil {
		panic(err)
	}

	enc := gob.NewEncoder(f)
	err = enc.Encode(blob)
	if err != nil {
		panic(err)
	}
	log.Printf("content stored to %s", fname)
}

func (ms *mapStore) ToBlob() []byte {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	b := &bytes.Buffer{}
	enc := gob.NewEncoder(b)
	err := enc.Encode(ms.store)
	if err != nil {
		panic(err)
	}
	return b.Bytes()
}

func (ps *pictoStore) ToBlob() []byte {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	b := &bytes.Buffer{}
	enc := gob.NewEncoder(b)
	err := enc.Encode(ps.store)
	if err != nil {
		panic(err)
	}
	return b.Bytes()
}
