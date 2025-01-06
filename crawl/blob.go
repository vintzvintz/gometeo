package crawl

import (
	"encoding/gob"
	"log"
	"os"
)

// utility type to store a MeteoContent without the ServeMux
type meteoBlob struct {
	Maps   MapStore
	Pictos PictoStore
}

func LoadContent(cacheFile string) *MeteoContent {
	f, err := os.Open(cacheFile)
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
	log.Printf("loaded maps & pictos from %s", cacheFile)
	mc := NewContent() // empty but non-nil
	mc.Update(blob.Maps, blob.Pictos)
	return mc
}

func StoreContent(cacheFile string, mc *MeteoContent) {
	f, err := os.Create(cacheFile)
	if err != nil {
		panic(err)
	}
	blob := meteoBlob{
		Maps:   mc.maps,
		Pictos: mc.pictos,
	}
	enc := gob.NewEncoder(f)
	err = enc.Encode(blob)
	if err != nil {
		panic(err)
	}
	log.Printf("content stored to %s", cacheFile)
}
