package content

import (
	_ "embed"
	"fmt"
	"net/http"
	"slices"

	"gometeo/mfmap"
)

// //go:embed status_template.html
//var tmpl string

func (ms *mapStore) Status() []mfmap.Stats {

	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	// sort keys by name for displaying maps in constant ordre
	// TODO : use stdlib functions available in go 1.23
	var names = make([]string, 0, len(ms.store))
	for k := range ms.store {
		names = append(names, k)
	}
	slices.Sort(names)

	stats := make([]mfmap.Stats, 0, len(ms.store))
	for _, name := range names {
		m := ms.store[name]
		stats = append(stats, m.Stats())
	}
	return stats
}

func (ms *mapStore) makeStatusHandler() http.HandlerFunc {
	return func(resp http.ResponseWriter, _ *http.Request) {
		resp.WriteHeader(http.StatusOK)
		for _, s := range ms.Status() {
			resp.Write([]byte(s.String()))
		}
	}
}

// return a human-readable string of a mfmap.UpdateMode internal code
func UpdateModeText(mode mfmap.UpdateMode) string {
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
