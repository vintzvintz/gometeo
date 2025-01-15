package mfmap

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

type atomicStats struct {
	lastUpdate atomic.Int64 // unix time, casted to time.Time by accessors
	lastHit    atomic.Int64 // unix time, casted to time.Time by accessors
	hitCount   atomic.Int64 // simple counter
}

type Stats struct {
	name       string
	lastUpdate time.Time
	lastHit    time.Time
	hitCount   int64
}

const (
	fastModeDuration = 3 * 24 * time.Hour // duration of fast update after last hit
	fastModeMaxAge   = 30 * time.Minute
	slowModeMaxAge   = 4 * time.Hour
)

func (m *MfMap) Stats() Stats {
	// return a static copy of atomic data

	return Stats{
		name:       m.Name(),
		hitCount:   m.stats.hitCount.Load(),
		lastHit:    time.Unix(m.stats.lastHit.Load(), 0),
		lastUpdate: time.Unix(m.stats.lastUpdate.Load(), 0),
	}
}

func (m *MfMap) LogUpdate() {
	now := time.Now().Unix()
	m.stats.lastUpdate.Store(now)
}

func (m *MfMap) LogHit() {
	now := time.Now().Unix()
	m.stats.lastHit.Store(now)
	m.stats.hitCount.Add(1)
}

func (m *MfMap) LastHit() time.Time {
	return time.Unix(m.stats.lastHit.Load(), 0)
}

func (m *MfMap) HitCount() int64 {
	return m.stats.hitCount.Load()
}

func (m *MfMap) NeedUpdate() bool {
	updateAge := time.Since(time.Unix(m.stats.lastUpdate.Load(), 0))
	hitAge := time.Since(time.Unix(m.stats.lastHit.Load(), 0))
	if hitAge < fastModeDuration {
		return updateAge > fastModeMaxAge
	}
	return updateAge > slowModeMaxAge
}

func (s Stats) String() string {
	return fmt.Sprintf("%s lastUpdate:%v lastHit:%v hitCount:%d\n",
		s.name, s.lastUpdate, s.lastHit, s.hitCount)
}

func (s Stats) Format(nameWidth int) string {
	// start with map name
	b := strings.Builder{}
	b.WriteString(s.name)
	// append space padding
	n := max(0, nameWidth-utf8.RuneCountInString(s.name))
	b.WriteString(strings.Repeat(" ", n))
	// append counters values
	b.WriteString(fmt.Sprintf(
		" lastUpdate:%v\tlastHit:%v", s.lastUpdate, s.lastHit))

	if s.hitCount > 0 {
		b.WriteString(fmt.Sprintf(" hitCount:%d", s.hitCount))
	}
	b.WriteByte('\n')
	return b.String()
}
