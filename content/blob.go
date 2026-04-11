package content

import (
	"encoding/gob"
	"fmt"
	"gometeo/geojson"
	"gometeo/mfmap"
	"log/slog"
	"os"
)

func init() {
	gob.Register(geojson.FloatTs{})
	gob.Register(geojson.IntTs{})
	gob.Register(geojson.FloatRangeTs{})
	gob.Register(geojson.IntRangeTs{})
}

// utility type to store a MeteoContent without the ServeMux and exposed fields
type meteoBlob struct {
	Maps   []*mfmap.MfMap
	Pictos []mfmap.Picto
}

// LoadBlob and SaveBlob are useful for dev and maintenance
func LoadBlob(fname string, cconf ContentConf, mconf mfmap.MapConf) *Meteo {
	// load and decode the whole blob
	f, err := os.Open(fname)
	if err != nil {
		slog.Error("LoadBlob open error", "err", err)
		return nil
	}
	dec := gob.NewDecoder(f)
	blob := meteoBlob{}
	err = dec.Decode(&blob)
	if err != nil {
		slog.Error("LoadBlob decode error", "err", err)
		return nil
	}
	mc := New(cconf)
	for _, m := range blob.Maps {
		m.Conf = mconf
		mc.maps.update(m, -1000, +1000)
	}
	for _, p := range blob.Pictos {
		mc.pictos.update(p)
	}
	mc.rebuildMux()
	slog.Info("loaded blob", "file", fname)
	return mc
}

// SaveBlob and LoadBlob are useful for dev and maintenance
func (mc *Meteo) SaveBlob(fname string) error {

	f, err := os.Create(fname)
	if err != nil {
		return fmt.Errorf("SaveBlob create %s: %w", fname, err)
	}
	defer f.Close()
	enc := gob.NewEncoder(f)

	blob := meteoBlob{
		Maps:   mc.maps.asSlice(),
		Pictos: mc.pictos.asSlice(),
	}
	err = enc.Encode(blob)
	if err != nil {
		return fmt.Errorf("SaveBlob encode: %w", err)
	}
	slog.Info("blob stored", "file", fname)
	return nil
}

// for load/save as binary blob
func (ps *pictoStore) asSlice() []mfmap.Picto {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	pictos := make([]mfmap.Picto, 0, len(ps.store))
	for name, img := range ps.store {
		pictos = append(pictos, mfmap.Picto{Name: name, Img: img})
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
