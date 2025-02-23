package mfmap

import (
	"sync"
	"testing"
	"time"

	"gometeo/appconf"
)

func TestAtomicRace(t *testing.T) {
	m := MfMap{}
	var wg sync.WaitGroup
	start := make(chan struct{})

	// prepare some goroutines
	const n = 50000
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start // wait for start signal
			m.MarkHit()
			m.MarkUpdate()
		}()
	}
	// unleach all goroutines for a massive attack !
	close(start)
	// wait for hitting to complete
	wg.Wait()

	if m.HitCount() != n {
		t.Errorf("concurrency issues...  got %d hits, want %d", m.HitCount(), n)
	}
	if time.Since(m.LastHit()).Abs() > time.Second {
		t.Errorf("last hit time not properly recorded")
	}
}

func TestLogUpdate(t *testing.T) {
	m := MfMap{}
	dur := m.DurationToUpdate()
	if dur > 0 {
		t.Error("DurationToUpdate() on zero-valued map expected negative")
	}
	m.MarkUpdate()
	dur = m.DurationToUpdate()
	if dur < 0 {
		t.Error("DurationToUpdate() on just-updated map must be positive")
	}
}

func TestNeedUpdate(t *testing.T) {
	r := appconf.UpdateRate()
	var tests = map[string]struct {
		update time.Time
		hit    time.Time
		want   bool
	}{
		"zero": {
			//update: 0,
			want:   true,
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
			update: time.Now().Add(-r.ColdMaxAge).Add(-30 * time.Second),   // last update 30 sec before cutoff
			want:   true,
		},
		"slow_false": {
			hit:    time.Now().Add(-r.HotDuration).Add(-10 * time.Second), // last hit 10 sec before fastmode cutoff
			update: time.Now().Add(-r.ColdMaxAge).Add(+30 * time.Second),   // last update 30 sec after cutoff
			want:   false,
		},
	}
	m := MfMap{}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			m.stats.lastUpdate.Store(test.update)
			m.stats.lastHit.Store(test.hit)
			got := m.DurationToUpdate() < 0
			if got != test.want {
				t.Errorf("DurationToUpdate()<0 got %v, want %v", got, test.want)
			}
		})
	}
}
