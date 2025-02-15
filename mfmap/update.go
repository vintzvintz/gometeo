package mfmap

import (
	"log"
	"sync/atomic"
	"time"
)

type atomicStats struct {
	lastUpdate atomic.Value // wraps a time.Time
	lastHit    atomic.Value // wraps a time.Time
	hitCount   atomic.Int64 // simple counter
}

type UpdateMode int

const (
	UPDATE_SLOW UpdateMode = iota
	UPDATE_FAST
)

const (
	//fastModeDuration = 30 * time.Minute
	//fastModeMaxAge   = 1 * time.Minute
	//slowModeMaxAge   = 5 * time.Minute
	fastModeDuration = 3 * 24 * time.Hour // duration of fast update after last hit
	fastModeMaxAge   = 30 * time.Minute
	slowModeMaxAge   = 4 * time.Hour
)

func (m *MfMap) MarkUpdate() {
	now := time.Now()
	m.stats.lastUpdate.Store(now)
}

func (m *MfMap) MarkHit() {
	now := time.Now()
	m.stats.lastHit.Store(now)
	m.stats.hitCount.Add(1)
}

func (m *MfMap) LastHit() time.Time {
	val := m.stats.lastHit.Load()
	if val == nil {
		return time.Time{}
	}
	t, ok := val.(time.Time)
	if !ok {
		log.Panicf("unexpected type")
	}
	return t
}

func (m *MfMap) LastUpdate() time.Time {
	val := m.stats.lastUpdate.Load()
	if val == nil {
		return time.Time{}
	}
	t, ok := val.(time.Time)
	if !ok {
		log.Panicf("unexpected type")
	}
	return t
}

func (m *MfMap) HitCount() int64 {
	return m.stats.hitCount.Load()
}

func (m *MfMap) UpdateMode() UpdateMode {
	hitAge := time.Since(m.LastHit())
	switch {
	case hitAge < fastModeDuration:
		return UPDATE_FAST
	default:
		return UPDATE_SLOW
	}
}

func (m *MfMap) DurationToUpdate() time.Duration {
	updateAge := time.Since(m.LastUpdate())
	switch m.UpdateMode() {
	case UPDATE_FAST:
		return fastModeMaxAge - updateAge
	case UPDATE_SLOW:
		fallthrough
	default:
		return slowModeMaxAge - updateAge
	}
}
