package mfmap

import (
	"sync"
	"testing"
)

func TestHitCountRace(t *testing.T) {
	m := MfMap{}
	var wg sync.WaitGroup
	start := make(chan struct{})

	// prepare some goroutines
	const n = 10000
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start // wait for start signal
			m.Hit()
		}()
	}
	// start all goroutines for massive hitting !
	close(start)
	// wait for hitting to complete
	wg.Wait()

	if m.HitCount() != n {
		t.Fatalf("concurrency issues...  got %d hits, want %d", m.HitCount(), n)
	}
}
