package schedule

import (
	"sync/atomic"
	"time"
)

type UpdateRates struct {
	HotDuration    time.Duration // map loses "hot" status after this delay
	HotMaxAge      time.Duration // update freq for "hot" maps
	ColdMaxAge     time.Duration // update rate for "cold" maps (default for maps never used)
	FailureBackoff time.Duration // delay before retrying a map after a fetch failure
}

type Stats struct {
	Rates        UpdateRates
	lastUpdate   atomic.Value // wraps a time.Time
	lastHit      atomic.Value // wraps a time.Time
	lastFailure  atomic.Value // wraps a time.Time
	lastClientIP atomic.Value // wraps a string
	hitCount     atomic.Int64 // simple counter
}

func (s *Stats) MarkUpdate() {
	now := time.Now()
	s.lastUpdate.Store(now)
	s.lastFailure.Store(time.Time{}) // clear any prior failure
}

func (s *Stats) MarkFailure() {
	s.lastFailure.Store(time.Now())
}

func (s *Stats) LastFailure() time.Time {
	return loadAsTime(&s.lastFailure)
}

func (s *Stats) MarkHit(clientIP string) {
	now := time.Now()
	s.lastHit.Store(now)
	s.lastClientIP.Store(clientIP)
	s.hitCount.Add(1)
}

func (s *Stats) LastHit() time.Time {
	return loadAsTime(&s.lastHit)
}

func (s *Stats) LastClientIP() string {
	val := s.lastClientIP.Load()
	if ip, ok := val.(string); ok {
		return ip
	}
	return ""
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

	var d time.Duration
	if hitAge < s.Rates.HotDuration {
		d = s.Rates.HotMaxAge - updateAge
	} else {
		d = s.Rates.ColdMaxAge - updateAge
	}
	// After a failure, hold off at least FailureBackoff before retrying,
	// even if the regular schedule says the map is overdue.
	if f := s.LastFailure(); !f.IsZero() {
		backoff := s.Rates.FailureBackoff - time.Since(f)
		if backoff > d {
			d = backoff
		}
	}
	return d
}

// CopyFrom copies hit stats from another Stats instance.
// Used during map merges to preserve hit tracking.
func (s *Stats) CopyFrom(other *Stats) {
	s.lastHit.Store(other.LastHit())
	s.lastClientIP.Store(other.LastClientIP())
	s.hitCount.Store(other.HitCount())
}
