package content

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"slices"
	"time"

	"gometeo/mfmap"
)

//go:embed status.html
var statusTemplate string

type Stats struct {
	Name       string
	Path       string
	LastUpdate time.Duration
	LastHit    time.Duration
	NextUpdate time.Duration
	UpdateMode string
	HitCount   int64
}

func getStats(m *mfmap.MfMap) Stats {
	var lh time.Duration
	if !m.LastHit().IsZero() {
		lh = time.Since(m.LastHit()).Round(time.Second)
	}
	s := Stats{
		Name:       m.Name(),
		Path:       m.Path(),
		HitCount:   m.HitCount(),
		UpdateMode: "-",
		LastHit:    lh,
		LastUpdate: time.Since(m.LastUpdate()).Round(time.Second),
		NextUpdate: m.DurationToUpdate().Round(time.Second),
	}
	if m.IsHot() {
		s.UpdateMode = "hot"
	}
	return s
}

func (s Stats) String() string {
	return fmt.Sprintf("%s mode:%s lastUpdate:%v lastHit:%v hitCount:%d\n",
		s.Name, s.UpdateMode, s.LastUpdate, s.LastHit, s.HitCount)
}

func (ms *mapStore) Status() []Stats {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	// sort keys by name for displaying maps in constant ordre
	// TODO : use stdlib functions available in go 1.23
	var names = make([]string, 0, len(ms.store))
	for k := range ms.store {
		names = append(names, k)
	}
	slices.Sort(names)

	stats := make([]Stats, 0, len(ms.store))
	for _, name := range names {
		m := ms.store[name]
		stats = append(stats, getStats(m))
	}
	return stats
}

func (ms *mapStore) makeStatusHandler() http.HandlerFunc {
	// compile template only once
	tmpl, err := template.New("").Parse(statusTemplate)
	if err != nil {
		panic(err) // error compiling status page template
	}
	// return closure having http.HandlerFunc signature
	return func(resp http.ResponseWriter, _ *http.Request) {
		d := struct {
			Stats     []Stats
			Updatable string
		}{
			Stats:     ms.Status(),
			Updatable: ms.updatable(),
		}
		b := &bytes.Buffer{}
		err := tmpl.Execute(b, d)
		if err != nil {
			log.Printf("statusHandler error: %s", err)
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}
		resp.Header().Add("X-Robots-Tag", "noindex, nofollow")
		resp.WriteHeader(http.StatusOK)
		_, err = io.Copy(resp, b)
		if err != nil {
			log.Printf("ignored send error: %s", err)
		}
	}
}
