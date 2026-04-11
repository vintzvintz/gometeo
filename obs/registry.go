// Package obs holds in-process observability state: counters and a small
// ring buffer of recent errors. The Registry is the single seam that future
// exporters (Prometheus, OpenTelemetry, JSON status endpoint) can read from
// without touching call sites.
//
// Extension path (do not reinvent — extend here):
//
//  1. Prometheus /metrics endpoint
//     Add obs/prometheus.go exposing a prometheus.Collector that reads the
//     atomic counters directly (no duplication). Register on the server mux
//     in server.makeMeteoHandler alongside /healthz. Counters stay the
//     source of truth; the collector is a read-only view.
//
//  2. JSON status endpoint (/statusse.json)
//     Snapshot is already the stable contract — marshal it directly.
//     Add obs/json.go or reuse from content/status.go. Same data the HTML
//     page renders, machine-readable for dashboards / curl checks.
//
//  3. OpenTelemetry metrics
//     Wrap the same atomic counters in otel Instruments via an
//     otel.Meter callback (async gauges/counters). Registry stays the
//     in-process state; OTel is just another exporter. Push interval
//     reads Snapshot().
//
//  4. Capture slog warnings/errors into the ring buffer
//     Implement a slog.Handler in obs/sloghandler.go that forwards records
//     at level >= Warn to RecordCrawlError (or a new RecordLogEvent).
//     Wire it as slog.SetDefault() in server.Start() before any goroutine
//     starts. Keeps call-site instrumentation optional — anything logged
//     ends up in the ring.
//
//  5. Per-map rolling stats (success rate, p50/p95 fetch latency)
//     Add a MapStats map keyed by path, protected by a sync.Map or RWMutex.
//     Recorder signatures already accept `path` — extend them, not replace.
//     Expose via Snapshot.MapStats []MapStat. Snapshot stays additive.
//
//  6. Persistence / crash recovery
//     Not planned. Ring is deliberately in-memory; restart resets it.
//     If ever needed, add obs/persist.go with a periodic gob dump loaded
//     in NewRegistry — do not spread persistence across recorders.
//
// Invariants to preserve when extending:
//   - Recorders must stay nil-safe and lock-free on the hot path (atomics only).
//   - Snapshot is the single read contract: add fields, never remove.
//   - Registry stays dependency-free (stdlib only). Exporters live in
//     sibling files and may pull in third-party libs.
//   - No package-level singletons — always inject *Registry via conf structs
//     so tests remain isolated.
package obs

import (
	"sync"
	"sync/atomic"
	"time"
)

// DefaultErrorRingSize is the number of recent errors kept in the ring buffer.
const DefaultErrorRingSize = 10

// Registry is the process-wide observability state. Construct one in the
// server entry point and inject it into crawl/content via their conf structs.
type Registry struct {
	startTime time.Time

	upstreamRequests atomic.Int64
	mapsFailed       atomic.Int64
	mapsServed       atomic.Int64
	pictosFailed     atomic.Int64
	pictosServed     atomic.Int64
	staticServed     atomic.Int64

	errors *errorRing
}

// NewRegistry returns a Registry with startTime set to now and an error ring
// of the default size.
func NewRegistry() *Registry {
	return NewRegistryWithSize(DefaultErrorRingSize)
}

// NewRegistryWithSize builds a Registry with a custom ring capacity (tests).
func NewRegistryWithSize(ringSize int) *Registry {
	return &Registry{
		startTime: time.Now(),
		errors:    newErrorRing(ringSize),
	}
}

// ErrorSource identifies what kind of operation produced an error event.
type ErrorSource string

const (
	SourceMap   ErrorSource = "map"
	SourcePicto ErrorSource = "picto"
	SourceCrawl ErrorSource = "crawl"
)

// ErrorEvent is one entry in the recent-errors ring buffer.
type ErrorEvent struct {
	Time   time.Time
	Source ErrorSource
	Target string
	Err    string
}

// Snapshot is a point-in-time view of the Registry state, safe to read
// outside any lock. Meant to be consumed by status handlers or exporters.
type Snapshot struct {
	StartTime     time.Time
	Uptime        time.Duration
	UpstreamRequests int64
	MapsFailed       int64
	MapsServed       int64
	PictosFailed     int64
	PictosServed     int64
	StaticServed     int64
	RecentErrors     []ErrorEvent // newest first
}

// RecordUpstreamRequest is called each time an HTTP request is actually
// sent to upstream (i.e. not served from the in-memory cache). The single
// counter intentionally does not split by resource kind — the crawl client
// is the only place that knows whether a call hit the network, and it
// does not know what the resource represents. Split by label later if a
// dashboard needs it.
func (r *Registry) RecordUpstreamRequest() {
	if r == nil {
		return
	}
	r.upstreamRequests.Add(1)
}

func (r *Registry) RecordMapFailed(path string, err error) {
	r.mapsFailed.Add(1)
	r.errors.push(ErrorEvent{
		Time:   time.Now(),
		Source: SourceMap,
		Target: path,
		Err:    errString(err),
	})
}

// RecordMapServed is called each time a map's JSON data is served.
// Nil-safe so handlers don't need to guard.
func (r *Registry) RecordMapServed() {
	if r == nil {
		return
	}
	r.mapsServed.Add(1)
}

// RecordPictoServed is called each time a picto is served.
// Nil-safe so handlers don't need to guard.
func (r *Registry) RecordPictoServed() {
	if r == nil {
		return
	}
	r.pictosServed.Add(1)
}

// RecordStaticServed is called each time an embedded static asset
// (JS, CSS, font, favicon, robots.txt) is served. Nil-safe.
func (r *Registry) RecordStaticServed() {
	if r == nil {
		return
	}
	r.staticServed.Add(1)
}

func (r *Registry) RecordPictoFailed(name string, err error) {
	r.pictosFailed.Add(1)
	r.errors.push(ErrorEvent{
		Time:   time.Now(),
		Source: SourcePicto,
		Target: name,
		Err:    errString(err),
	})
}

// RecordCrawlError records a non-target-specific crawl error (e.g. context
// expiration). Target may be empty.
func (r *Registry) RecordCrawlError(target string, err error) {
	r.errors.push(ErrorEvent{
		Time:   time.Now(),
		Source: SourceCrawl,
		Target: target,
		Err:    errString(err),
	})
}

// Snapshot returns a consistent read of the registry state.
func (r *Registry) Snapshot() Snapshot {
	return Snapshot{
		StartTime:        r.startTime,
		Uptime:           time.Since(r.startTime),
		UpstreamRequests: r.upstreamRequests.Load(),
		MapsFailed:       r.mapsFailed.Load(),
		MapsServed:       r.mapsServed.Load(),
		PictosFailed:     r.pictosFailed.Load(),
		PictosServed:     r.pictosServed.Load(),
		StaticServed:     r.staticServed.Load(),
		RecentErrors:     r.errors.snapshot(),
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// errorRing is a fixed-size ring buffer of ErrorEvents, newest-last.
// Guarded by a single mutex; writes are rare (only on errors).
type errorRing struct {
	mu   sync.Mutex
	buf  []ErrorEvent
	next int   // index of next write
	full bool  // true once buf has been filled at least once
	size int
}

func newErrorRing(size int) *errorRing {
	if size <= 0 {
		size = DefaultErrorRingSize
	}
	return &errorRing{
		buf:  make([]ErrorEvent, size),
		size: size,
	}
}

func (r *errorRing) push(ev ErrorEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buf[r.next] = ev
	r.next = (r.next + 1) % r.size
	if r.next == 0 {
		r.full = true
	}
}

// snapshot returns ring contents newest-first.
func (r *errorRing) snapshot() []ErrorEvent {
	r.mu.Lock()
	defer r.mu.Unlock()

	var count int
	if r.full {
		count = r.size
	} else {
		count = r.next
	}
	out := make([]ErrorEvent, count)
	// walk backwards from the most recent write
	for i := 0; i < count; i++ {
		idx := (r.next - 1 - i + r.size) % r.size
		out[i] = r.buf[idx]
	}
	return out
}
