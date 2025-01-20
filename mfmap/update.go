package mfmap

import (
	"sync/atomic"
	"time"
)

type atomicStats struct {
	lastUpdate atomic.Int64 // unix time, casted to time.Time by accessors
	lastHit    atomic.Int64 // unix time, casted to time.Time by accessors
	hitCount   atomic.Int64 // simple counter
}

type UpdateMode int

const (
	UPDATE_SLOW UpdateMode = iota
	UPDATE_FAST
)

const (
	fastModeDuration = 3 * 24 * time.Hour // duration of fast update after last hit
	fastModeMaxAge   = 600 * time.Second
	slowModeMaxAge   = 3600 * time.Second
	//fastModeMaxAge = 30 * time.Minute
	//slowModeMaxAge = 4 * time.Hour
)

func (m *MfMap) MarkUpdate() {
	now := time.Now().Unix()
	m.stats.lastUpdate.Store(now)
}

func (m *MfMap) MarkHit() {
	now := time.Now().Unix()
	m.stats.lastHit.Store(now)
	m.stats.hitCount.Add(1)
}

func (m *MfMap) LastHit() time.Time {
	return time.Unix(m.stats.lastHit.Load(), 0)
}

func (m *MfMap) LastUpdate() time.Time {
	return time.Unix(m.stats.lastUpdate.Load(), 0)
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
