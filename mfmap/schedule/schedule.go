package schedule

import (
	"sync/atomic"
	"time"
)

type UpdateRates struct {
	HotDuration time.Duration // map loses "hot" status after this delay
	HotMaxAge   time.Duration // update freq for "hot" maps
	ColdMaxAge  time.Duration // update rate for "cold" maps (default for maps never used)
}

type Stats struct {
	Rates      UpdateRates
	lastUpdate atomic.Value // wraps a time.Time
	lastHit    atomic.Value // wraps a time.Time
	hitCount   atomic.Int64 // simple counter
}

func (s *Stats) MarkUpdate() {
	now := time.Now()
	s.lastUpdate.Store(now)
}

func (s *Stats) MarkHit() {
	now := time.Now()
	s.lastHit.Store(now)
	s.hitCount.Add(1)
}

func (s *Stats) LastHit() time.Time {
	return loadAsTime(&s.lastHit)
}

func (s *Stats) LastUpdate() time.Time {
	return loadAsTime(&s.lastUpdate)
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

func (s *Stats) HitCount() int64 {
	return s.hitCount.Load()
}

func (s *Stats) IsHot() bool {
	hitAge := time.Since(s.LastHit())
	return hitAge < s.Rates.HotDuration
}

func (s *Stats) DurationToUpdate() time.Duration {
	hitAge := time.Since(s.LastHit())
	updateAge := time.Since(s.LastUpdate())

	if hitAge < s.Rates.HotDuration {
		return s.Rates.HotMaxAge - updateAge
	} else {
		return s.Rates.ColdMaxAge - updateAge
	}
}

// CopyFrom copies hit stats from another Stats instance.
// Used during map merges to preserve hit tracking.
func (s *Stats) CopyFrom(other *Stats) {
	s.lastHit.Store(other.LastHit())
	s.hitCount.Store(other.HitCount())
}
