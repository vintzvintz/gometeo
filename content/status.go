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

//go:embed status_template.html
var statusTemplate string

type Stats struct {
	Name       string
	LastUpdate time.Time
	LastHit    time.Time
	NextUpdate time.Duration
	UpdateMode string
	HitCount   int64
}

func getStats(m *mfmap.MfMap) Stats {
	return Stats{
		Name:       m.Name(),
		HitCount:   m.HitCount(),
		UpdateMode: updateModeText(m.UpdateMode()),
		LastHit:    m.LastHit(),
		LastUpdate: m.LastUpdate(),
		NextUpdate: m.DurationToUpdate().Round(time.Second),
	}
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

/*
	func (ms *mapStore) makeBasicStatusHandler() http.HandlerFunc {
		return func(resp http.ResponseWriter, _ *http.Request) {
			resp.WriteHeader(http.StatusOK)
			for _, s := range ms.Status() {
				resp.Write([]byte(s.String()))
			}
		}
	}
*/
func (ms *mapStore) makeStatusHandler() http.HandlerFunc {
	// compile template only once
	tmpl, err := template.New("").Parse(statusTemplate)
	if err != nil {
		panic(err) // error compiling status page template
	}
	// return closure having http.HandlerFunc signature
	return func(resp http.ResponseWriter, _ *http.Request) {
		b := &bytes.Buffer{}
		err := tmpl.Execute(b, ms.Status())
		if err != nil {
			log.Printf("statusHandler error: %s", err)
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = io.Copy(resp, b)
		if err != nil {
			log.Printf("ignored send error: %s", err)
		}
	}
}

// return a human-readable string of a mfmap.UpdateMode internal code
func updateModeText(mode mfmap.UpdateMode) string {
	var updateModeText = map[mfmap.UpdateMode]string{
		mfmap.UPDATE_FAST: "fast",
		mfmap.UPDATE_SLOW: "slow",
	}
	s, ok := updateModeText[mode]
	if !ok {
		return fmt.Sprintf("unknown mode (%d)", mode)
	}
	return s
}

/*
func FormatMapStat(s mfmap.Stats, nameWidth int) string {
	// start with map name
	b := strings.Builder{}
	b.WriteString(s.Name)
	// append space padding
	n := max(0, nameWidth-utf8.RuneCountInString(s.Name))
	b.WriteString(strings.Repeat(" ", n))
	// append counters values
	b.WriteString(fmt.Sprintf(
		" lastUpdate:%v\tlastHit:%v", s.LastUpdate, s.LastHit))

	if s.HitCount > 0 {
		b.WriteString(fmt.Sprintf(" hitCount:%d", s.HitCount))
	}
	b.WriteByte('\n')
	return b.String()
}
*/
