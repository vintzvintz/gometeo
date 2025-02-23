package mfmap

import (
	"sync/atomic"
	"time"

	"gometeo/appconf"
)

type atomicStats struct {
	lastUpdate atomic.Value // wraps a time.Time
	lastHit    atomic.Value // wraps a time.Time
	hitCount   atomic.Int64 // simple counter
}

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
	return loadAsTime(&m.stats.lastHit)
}

func (m *MfMap) LastUpdate() time.Time {
	return loadAsTime(&m.stats.lastUpdate)
}

func loadAsTime(a *atomic.Value) time.Time {
	val := a.Load()
	switch t := val.(type) {
	case time.Time:
		return t
	default:
		return time.Time{}
	}
}

func (m *MfMap) HitCount() int64 {
	return m.stats.hitCount.Load()
}

func (m *MfMap) IsHot() bool {
	r := appconf.UpdateRate()
	hitAge := time.Since(m.LastHit())
	return hitAge < r.HotDuration 
}

func (m *MfMap) DurationToUpdate() time.Duration {
	r := appconf.UpdateRate()
	hitAge := time.Since(m.LastHit())
	updateAge := time.Since(m.LastUpdate())

	if hitAge < r.HotDuration {
		return r.HotMaxAge - updateAge
	} else {
		return r.ColdMaxAge - updateAge
	}
}
