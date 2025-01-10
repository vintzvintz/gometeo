package mfmap

import (
	"sync/atomic"
	"time"
)

type Stats struct {
	lastUpdate time.Time
	lastHit    time.Time
	hitCount   atomic.Int64
}

func (m *MfMap) Hit() {
	_ = m.stats.lastHit
	_ = m.stats.lastUpdate
	m.stats.hitCount.Add(1)
}

func (m *MfMap) HitCount() int64 {
	return m.stats.hitCount.Load()
}
