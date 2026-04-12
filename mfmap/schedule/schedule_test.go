package schedule

import (
	"sync"
	"testing"
	"time"
)

func TestAtomicRace(t *testing.T) {
	s := Stats{}
	var wg sync.WaitGroup
	start := make(chan struct{})

	// prepare some goroutines
	const n = 50000
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start // wait for start signal
			s.MarkHit("")
			s.MarkUpdate()
		}()
	}
	// unleach all goroutines for a massive attack !
	close(start)
	// wait for hitting to complete
	wg.Wait()

	if s.HitCount() != n {
		t.Errorf("concurrency issues...  got %d hits, want %d", s.HitCount(), n)
	}
	if time.Since(s.LastHit()).Abs() > time.Second {
		t.Errorf("last hit time not properly recorded")
	}
}

var testRates = UpdateRates{
	HotDuration:    72 * time.Hour,
	HotMaxAge:      60 * time.Minute,
	ColdMaxAge:     240 * time.Minute,
	FailureBackoff: 30 * time.Minute,
}

func TestLogUpdate(t *testing.T) {
	s := Stats{Rates: testRates}
	dur := s.DurationToUpdate()
	if dur > 0 {
		t.Error("DurationToUpdate() on zero-valued map expected negative")
	}
	s.MarkUpdate()
	dur = s.DurationToUpdate()
	if dur < 0 {
		t.Error("DurationToUpdate() on just-updated map must be positive")
	}
}

func TestFailureBackoff(t *testing.T) {
	r := testRates
	// A map with a stale lastUpdate would normally be due immediately.
	// After MarkFailure, it must not be due until FailureBackoff has elapsed.
	s := Stats{Rates: r}
	s.lastUpdate.Store(time.Now().Add(-2 * r.ColdMaxAge)) // very overdue
	if d := s.DurationToUpdate(); d > 0 {
		t.Fatalf("precondition: expected overdue map, got d=%v", d)
	}
	s.MarkFailure()
	if d := s.DurationToUpdate(); d <= 0 {
		t.Errorf("after MarkFailure: expected backoff > 0, got d=%v", d)
	}

	// Simulating a failure further in the past than FailureBackoff should
	// let the map become due again.
	s.lastFailure.Store(time.Now().Add(-r.FailureBackoff - time.Minute))
	if d := s.DurationToUpdate(); d > 0 {
		t.Errorf("after backoff expired: expected due, got d=%v", d)
	}

	// A successful MarkUpdate must clear the failure state.
	s.MarkFailure()
	s.MarkUpdate()
	if !s.LastFailure().IsZero() {
		t.Errorf("MarkUpdate must clear lastFailure, got %v", s.LastFailure())
	}
}

func TestNeedUpdate(t *testing.T) {
	r := testRates
	var tests = map[string]struct {
		update time.Time
		hit    time.Time
		want   bool
	}{
		"zero": {
			//update: 0,
			want: true,
		},
		"now": {
			update: time.Now(),
			want:   false,
		},
		"fast_true": {
			hit:    time.Now().Add(-r.HotDuration).Add(10 * time.Second), // last hit 10 sec after fastmode cutoff
			update: time.Now().Add(-r.HotMaxAge).Add(-30 * time.Second),  // last update 30 sec before cutoff
			want:   true,
		},
		"fast_false": {
			hit:    time.Now().Add(-r.HotDuration).Add(+10 * time.Second), // last hit 10 sec after fastmode cutoff
			update: time.Now().Add(-r.HotMaxAge).Add(+30 * time.Second),   // last update 30 sec after cutoff
			want:   false,
		},
		"slow_true": {
			hit:    time.Now().Add(-r.HotDuration).Add(-10 * time.Second), // last hit 10 sec before fastmode cutoff
			update: time.Now().Add(-r.ColdMaxAge).Add(-30 * time.Second),  // last update 30 sec before cutoff
			want:   true,
		},
		"slow_false": {
			hit:    time.Now().Add(-r.HotDuration).Add(-10 * time.Second), // last hit 10 sec before fastmode cutoff
			update: time.Now().Add(-r.ColdMaxAge).Add(+30 * time.Second),  // last update 30 sec after cutoff
			want:   false,
		},
	}
	s := Stats{Rates: testRates}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			s.lastUpdate.Store(test.update)
			s.lastHit.Store(test.hit)
			got := s.DurationToUpdate() < 0
			if got != test.want {
				t.Errorf("DurationToUpdate()<0 got %v, want %v", got, test.want)
			}
		})
	}
}
