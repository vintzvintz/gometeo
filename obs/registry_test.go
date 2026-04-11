package obs

import (
	"errors"
	"sync"
	"testing"
)

func TestCountersRecord(t *testing.T) {
	r := NewRegistry()
	r.RecordUpstreamRequest()
	r.RecordUpstreamRequest()
	r.RecordUpstreamRequest()
	r.RecordMapFailed("/c", errors.New("boom"))
	r.RecordPictoFailed("p2j", errors.New("nope"))

	s := r.Snapshot()
	if s.UpstreamRequests != 3 {
		t.Errorf("UpstreamRequests = %d, want 3", s.UpstreamRequests)
	}
	if s.MapsFailed != 1 {
		t.Errorf("MapsFailed = %d, want 1", s.MapsFailed)
	}
	if s.PictosFailed != 1 {
		t.Errorf("PictosFailed = %d, want 1", s.PictosFailed)
	}
	if s.Uptime <= 0 {
		t.Errorf("Uptime = %v, want > 0", s.Uptime)
	}
}

func TestErrorRingNewestFirst(t *testing.T) {
	r := NewRegistryWithSize(3)
	r.RecordMapFailed("/a", errors.New("e1"))
	r.RecordMapFailed("/b", errors.New("e2"))
	r.RecordMapFailed("/c", errors.New("e3"))

	errs := r.Snapshot().RecentErrors
	if len(errs) != 3 {
		t.Fatalf("len = %d, want 3", len(errs))
	}
	wantOrder := []string{"/c", "/b", "/a"}
	for i, want := range wantOrder {
		if errs[i].Target != want {
			t.Errorf("errs[%d].Target = %q, want %q", i, errs[i].Target, want)
		}
	}
}

func TestErrorRingWrapAround(t *testing.T) {
	r := NewRegistryWithSize(3)
	for _, tgt := range []string{"/1", "/2", "/3", "/4", "/5"} {
		r.RecordMapFailed(tgt, errors.New("x"))
	}
	errs := r.Snapshot().RecentErrors
	if len(errs) != 3 {
		t.Fatalf("len = %d, want 3", len(errs))
	}
	want := []string{"/5", "/4", "/3"}
	for i, w := range want {
		if errs[i].Target != w {
			t.Errorf("errs[%d] = %q, want %q", i, errs[i].Target, w)
		}
	}
}

func TestErrorRingPartial(t *testing.T) {
	r := NewRegistryWithSize(5)
	r.RecordMapFailed("/a", errors.New("x"))
	r.RecordPictoFailed("p", errors.New("y"))

	errs := r.Snapshot().RecentErrors
	if len(errs) != 2 {
		t.Fatalf("len = %d, want 2", len(errs))
	}
	if errs[0].Source != SourcePicto || errs[1].Source != SourceMap {
		t.Errorf("source order = [%s, %s], want [picto, map]", errs[0].Source, errs[1].Source)
	}
}

func TestConcurrentRecorders(t *testing.T) {
	r := NewRegistryWithSize(50)
	const goroutines = 20
	const per = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < per; j++ {
				r.RecordUpstreamRequest()
				r.RecordMapFailed("/y", errors.New("e"))
			}
		}()
	}
	wg.Wait()

	s := r.Snapshot()
	if s.UpstreamRequests != goroutines*per {
		t.Errorf("UpstreamRequests = %d, want %d", s.UpstreamRequests, goroutines*per)
	}
	if s.MapsFailed != goroutines*per {
		t.Errorf("MapsFailed = %d, want %d", s.MapsFailed, goroutines*per)
	}
	if len(s.RecentErrors) != 50 {
		t.Errorf("RecentErrors len = %d, want 50", len(s.RecentErrors))
	}
}
