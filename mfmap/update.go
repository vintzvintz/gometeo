package mfmap

import (
	"sync/atomic"
	"time"
)

type Stats struct {
	lastUpdate atomic.Int64 // unix time, casted to time.Time by accessors
	lastHit    atomic.Int64 // unix time, casted to time.Time by accessors
	hitCount   atomic.Int64 // simple counter
}

const (
	fastModeDuration = 3 * 24 * time.Hour // duration of fast update after last hit
	fastModeMaxAge   = 30 * time.Minute
	slowModeMaxAge   = 4 * time.Hour
)

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

func (m *MfMap) ShouldUpdate() bool {
	updateAge := time.Since(time.Unix(m.stats.lastUpdate.Load(), 0))
	hitAge := time.Since(time.Unix(m.stats.lastHit.Load(), 0))
	if hitAge < fastModeDuration {
		return updateAge > fastModeMaxAge
	}
	return updateAge > slowModeMaxAge
}
