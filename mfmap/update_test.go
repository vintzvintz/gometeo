package mfmap

import (
	"sync"
	"testing"
	"time"
)

func TestHitCountRace(t *testing.T) {
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
			m.LogHit()
		}()
	}
	// start all goroutines for massive hitting !
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
	before := m.ShouldUpdate()
	if !before {
		t.Error("ShouldUpdate() on zero-valued map must be true")
	}
	m.LogUpdate()
	after := m.ShouldUpdate()
	if after {
		t.Error("ShouldUpdate() on just-updated map must be false")
	}
}

func TestShouldUpdate(t *testing.T) {

	var tests = map[string]struct {
		update int64
		hit    int64
		want   bool
	}{
		"zero": {
			update: 0,
			want:   true,
		},
		"now": {
			update: time.Now().Unix(),
			want:   false,
		},
		"fast_true": {
			hit:    time.Now().Add(-fastModeDuration).Unix() + 10, // last hit 10 sec after fastmode cutoff
			update: time.Now().Add(-fastModeMaxAge).Unix() - 30,   // last update 30 sec before cutoff
			want:   true,
		},
		"fast_false": {
			hit:    time.Now().Add(-fastModeDuration).Unix() + 10, // last hit 10 sec after fastmode cutoff
			update: time.Now().Add(-fastModeMaxAge).Unix() + 30,   // last update 30 sec after cutoff
			want:   false,
		},
		"slow_true": {
			hit:    time.Now().Add(-fastModeDuration).Unix() - 10, // last hit 10 sec before fastmode cutoff
			update: time.Now().Add(-slowModeMaxAge).Unix() - 30,   // last update 30 sec before cutoff
			want:   true,
		},
		"slow_false": {
			hit:    time.Now().Add(-fastModeDuration).Unix() - 10, // last hit 10 sec before fastmode cutoff
			update: time.Now().Add(-slowModeMaxAge).Unix() + 30,   // last update 30 sec after cutoff
			want:   false,
		},
	}
	m := MfMap{}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			m.stats.lastUpdate.Store(test.update)
			m.stats.lastHit.Store(test.hit)
			got := m.ShouldUpdate()
			if got != test.want {
				t.Errorf("ShouldUpdate() got %v, want %v", got, test.want)
			}
		})
	}
}
