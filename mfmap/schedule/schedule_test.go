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
			s.MarkHit()
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
	HotDuration: 72 * time.Hour,
	HotMaxAge:   60 * time.Minute,
	ColdMaxAge:  240 * time.Minute,
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
