# Gometeo Work Plan

Detailed implementation plan for improving and future-proofing gometeo.
Each task is self-contained with enough context to start in a fresh session.

Baseline state: all tests pass (`go test ./...` and `go test -race ./...`), `go vet` clean. Go 1.22, two dependencies (x/net, etree), deployed via Docker+Traefik on gometeo.vintz.fr.

---

## Phase 1 — Bug fixes & hygiene

Quick wins, no architectural changes. Can be done as individual commits.

### 1.1 Fix slice pre-allocation bug in `content/blob.go`

**File:** `content/blob.go:66-74` — `pictoStore.asSlice()`

**Bug:** `make([]Picto, len(ps.store))` creates a slice with `len` zero-valued elements, then `append()` adds real elements after them. The result has N empty Picto entries followed by N real ones.

**Fix:** Change line 69 from:
```go
pictos := make([]Picto, len(ps.store))
```
to:
```go
pictos := make([]Picto, 0, len(ps.store))
```

**Compare with:** `mapStore.asSlice()` at line 77-85 in the same file, which does it correctly.

**Impact:** `SaveBlob`/`LoadBlob` (dev/debug tool for caching crawled data to disk) will produce correct output. Currently, loading a blob creates ghost empty pictos.

**Test:** There are no tests for `content/` yet (see task 3.1). Manually verify with `./gometeo -oneshot -limit 5 -fastupdate` — the blob save/load cycle should round-trip cleanly.

---

### 1.2 Upgrade Go version from 1.22 to 1.24+

**Current state:** `go.mod` declares `go 1.22`, local toolchain is 1.22.2, latest stable is 1.26.2. Go 1.22 is out of support (no more security patches).

**What to do:**
1. Install a current Go toolchain (1.24+ at minimum for security support)
2. Update `go.mod`: change `go 1.22` to the new version
3. Update `Dockerfile` line 1: change `golang:1.22.11-bookworm` to matching version
4. Run `go mod tidy` to update go.sum
5. Run `go test -race ./...` and `go vet ./...`
6. Check if any new `go vet` warnings appear (newer Go versions add checks)

**Breaking changes to watch for:**
- Go 1.23: `range over int` became stable, `maps` and `slices` packages moved to stdlib — there's a commented TODO about this in `mfmap/merge.go:76`
- Go 1.24: no breaking changes for this codebase

**Note:** The `docker-compose.yml` line 6 hardcodes the binary path as `/build/gometeo` — this is unaffected by Go version changes.

---

### 1.3 Delete dead code in `mfmap/merge.go`

**File:** `mfmap/merge.go` — the entire file is 83 lines of commented-out code (old merge implementation superseded by `geojson/merge.go`).

**What to do:** Delete the file entirely. The code is preserved in git history if ever needed.

**Verify:** `go build ./...` and `go test ./...` still pass (the file contributes nothing since it's all comments).

---

### 1.4 Add timeout to the forever-update fetch loop

**File:** `crawl/crawler.go:62-71`

**Current code:**
```go
for {
    time.Sleep(fetchInterval * time.Second)
    path := mc.Updatable()
    if path == "" {
        continue
    }
    // TODO: add timeout
    <-mc.Receive(cr.Fetch(path, 1))
}
```

**Problem:** If upstream hangs (DNS timeout, TCP stall, HTTP body never closes), the crawler goroutine blocks forever on `Receive`. No more maps get updated.

**Fix:** Wrap the fetch+receive in a context with timeout:
```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
defer cancel()
// Use a goroutine + select to enforce the timeout
done := mc.Receive(cr.Fetch(path, 1))
select {
case <-done:
    // OK
case <-ctx.Done():
    log.Printf("fetch timeout for %s", path)
}
```

**Consideration:** `Fetch()` and the underlying `Client.Get()` don't accept a context yet. The simplest approach is to add a `context.Context` parameter to `Client.Get()` (in `crawl/client.go`) and propagate it through `Fetch()` and `getMap()`. Alternatively, wrap just the receive with a timer as shown above — simpler but doesn't cancel the in-flight HTTP request.

**The simpler timer-based approach is recommended first**, and context propagation can be a follow-up if needed.

---

### 1.5 Handle ignored `io.Copy` errors in static file handler

**File:** `static/static.go:99`
```go
io.Copy(w, f)  // return value ignored
```

**Fix:** Log the error (consistent with how `mfmap/handlers.go` already does it at lines 42-44, 61-63, 81-83):
```go
if _, err := io.Copy(w, f); err != nil {
    log.Printf("favicon send error: %s", err)
}
```

**Note:** The handlers in `mfmap/handlers.go` already handle this correctly — they log `"ignored send error"`. This is just the one remaining spot.

---

## Phase 2 — Docker & deployment

### 2.1 Multi-stage Dockerfile

**File:** `Dockerfile` (22 lines)

**Current state:** Single-stage build using `golang:1.22.11-bookworm` as the runtime image. The final image contains the full Go toolchain, apt packages, etc (~800MB+).

**Target:** Two-stage build. Build stage compiles the binary, final stage uses a minimal base.

**Recommended approach:**
```dockerfile
# Build stage
FROM golang:<version>-bookworm AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o gometeo

# Runtime stage
FROM gcr.io/distroless/static-debian12
COPY --from=builder /build/gometeo /gometeo
EXPOSE 1051
ENTRYPOINT ["/gometeo"]
```

**Why distroless over scratch:** Includes CA certificates (needed for HTTPS to meteofrance.com) and timezone data (the app uses `time` package and docker-compose sets `TZ=Europe/Paris`).

**Why `CGO_ENABLED=0`:** The codebase uses no CGo. This produces a fully static binary that runs on any Linux.

**Update `docker-compose.yml` line 6:** Change command from `["/build/gometeo", ...]` to `["/gometeo", ...]`.

**Expected result:** Image size drops from ~800MB to ~15-20MB.

---

### 2.2 Add health check endpoint

**Current state:** No health endpoint. Docker has no way to know if the app is healthy. If the crawler hangs or the mux is nil, the container stays "running" but serves nothing.

**Implementation — two parts:**

**Part A: Add `/health` endpoint in `server/server.go`**

In `makeMeteoHandler()` (line 84), register a health handler on the outer mux before the catch-all:
```go
mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("ok"))
})
```

This should be registered before `mux.Handle("/", mc)` so it takes priority. It doesn't need to go through the Meteo handler chain.

**Part B: Add healthcheck to `docker-compose.yml`**

```yaml
services:
  app:
    # ... existing config ...
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:1051/health"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 60s
```

**Note on `wget`:** distroless images don't include wget/curl. Two options:
- Use a distroless image variant with busybox, OR
- Build a tiny Go health-check binary in the Dockerfile and COPY it alongside the main binary, OR
- Use the app itself: add a `-healthcheck` CLI flag that makes a GET to localhost and exits 0/1

The CLI flag approach is cleanest: add a flag in `appconf` that, when set, makes the binary do a single HTTP GET to `http://localhost:<addr>/health` and exit. Then the docker healthcheck becomes `["/gometeo", "-healthcheck"]`.

---

### 2.3 Make configuration overridable via environment variables

**File:** `appconf/conf.go`

**Current state:** All config comes from CLI flags (lines 75-103). Constants like `DEFAULT_ADDR`, `UPSTREAM_ROOT`, and update intervals are hardcoded. There's a `// TODO: refactor into env var` comment at line 11.

**What to change:** For each constant that should be configurable at deploy time, check for an env var before falling back to the default. The most useful ones:

| Env var | Constant | Line | Current value |
|---------|----------|------|---------------|
| `GOMETEO_ADDR` | `DEFAULT_ADDR` | 12 | `:1051` |
| `GOMETEO_UPSTREAM` | `UPSTREAM_ROOT` | 14 | `https://meteofrance.com` |

**Approach:** In `getOpts()`, use `os.Getenv()` to set flag defaults:
```go
defaultAddr := DEFAULT_ADDR
if env := os.Getenv("GOMETEO_ADDR"); env != "" {
    defaultAddr = env
}
f.StringVar(&opts.Addr, "addr", defaultAddr, "listening server address")
```

CLI flags still take priority (standard Go flag behavior: explicit flag > default). This keeps backward compatibility.

**Update intervals** (`normalHotDuration`, etc.) are less critical to expose — the `-fastupdate` flag already provides a dev mode. Consider this optional.

---

## Phase 3 — Test coverage

### 3.1 Add tests for `content/` package

**Current state:** `content/` has zero test files. This package contains `Meteo`, `mapStore`, `pictoStore`, mux rebuild logic, and the `Receive()` pipeline — all critical runtime code.

**What needs tests:**

**A. `mapStore` basic operations** (`content/content.go:143-158`)
- `update()` should add a new map, retrieve it, and build breadcrumbs
- `update()` on existing path should call `Merge()` and preserve stats
- `updatable()` should return the map with the oldest update time
- `buildBreadcrumbs()` should produce correct parent chains

**B. `pictoStore` basic operations**
- `update()` should store and overwrite pictos
- `asSlice()` should return all stored pictos (and no ghost empty ones — validates fix from 1.1)

**C. `Receive()` pipeline** (`content/content.go:69-98`)
- Send maps and pictos through channels, verify they appear in the store
- Verify `done` channel closes after both input channels close

**D. `meteoMux` hot-swap** (`content/content.go:250-260`)
- Concurrent reads during `setMux()` should not panic
- After `rebuildMux()`, new routes should be reachable

**Helper code:** `testutils/` already provides `MockHandlers()` and `MakeMfMap()` — use these to create test fixtures.

**File to create:** `content/content_test.go`

**Test structure:**
```go
func TestMapStoreUpdate(t *testing.T) { ... }
func TestMapStoreUpdatable(t *testing.T) { ... }
func TestPictoStoreRoundTrip(t *testing.T) { ... }
func TestReceivePipeline(t *testing.T) { ... }
func TestConcurrentMuxSwap(t *testing.T) { ... }
```

**Run with:** `go test -race ./content/...`

---

### 3.2 Add integration test for crawler-to-handler pipeline

**Goal:** Test the full path: mock upstream HTTP server → crawler → content store → HTTP handler response.

**Approach:**
1. Use `testutils.MockHandlers()` to create a fake upstream serving test fixtures from `test_data/`
2. Create a `crawl.Crawler` pointed at the mock server
3. Call `content.New()` + `Receive(crawler.Fetch(...))`
4. Wait for `done` channel
5. Make HTTP requests to the resulting `Meteo` handler and verify responses

**File to create:** `server/integration_test.go` or a top-level `integration_test.go`

**Note:** The `testutils/handlers.go` file already sets up mock HTTP handlers for most upstream endpoints. The test needs to wire them together with a real crawler and content pipeline.

---

### 3.3 Set up CI with GitHub Actions

**File to create:** `.github/workflows/ci.yml`

**Minimal pipeline:**
```yaml
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - run: go vet ./...
      - run: go test -race ./...
```

**Why:** Prevents regressions. Currently there's no CI — tests only run when someone remembers to.

**Optional additions:**
- `golangci-lint` for static analysis
- Docker build step to verify the Dockerfile
- Cache `go mod download` for speed

---

## Phase 4 — Hardening

Lower priority. Do these when the above are done.

### 4.1 Replace panics in `SaveBlob` with returned errors

**File:** `content/blob.go:46-63`

**Current code:** `SaveBlob()` panics on file creation error (line 50) and encoding error (line 60). The function signature is `func (mc *Meteo) SaveBlob(fname string)` with no error return.

**Fix:** Change signature to `func (mc *Meteo) SaveBlob(fname string) error`, return errors instead of panicking.

**Callers:** Only `server/server.go:57` — `c.SaveBlob(cacheFile)`. Update to check the returned error.

**Note:** `LoadBlob` already handles errors gracefully (returns nil).

---

### 4.2 Migrate to structured logging (`log/slog`)

**Current state:** All logging uses `log.Printf` / `log.Println` / `log.Fatal` throughout the codebase. No log levels, no structured fields.

**What to do:** Replace `log` with `log/slog` (stdlib since Go 1.21). This gives:
- Log levels (Debug, Info, Warn, Error)
- Structured key-value fields
- JSON output option (useful for log aggregation)

**Example migration:**
```go
// Before
log.Printf("getMap() '%s'", path)

// After
slog.Info("getMap", "path", path)
```

**Scope:** Touch every `.go` file with `log.Printf`. This is a large mechanical change — do it in one commit per package or one big commit. Don't mix with other changes.

**Priority:** Low — the current logging works fine. Do this if you plan to add monitoring/alerting.

---

### 4.3 Add security headers middleware

**Current state:** No security headers set by the Go server. Traefik may or may not add them.

**Check first:** Inspect the Traefik config on the production host. If Traefik already sets these headers (via `traefik.http.middlewares.*` labels or file config), skip this task.

**If not handled by Traefik, add middleware in `server/server.go`:**
```go
func withSecurityHeaders(h http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        h.ServeHTTP(w, r)
    })
}
```

Add it to the middleware chain in `makeMeteoHandler()` (line 84-91).

---

### 4.4 Optimize breadcrumb rebuild

**File:** `content/content.go:155-158`

**Current code:** On every map update, `buildBreadcrumbs()` is called for ALL maps in the store. At 100 maps this means 100 breadcrumb rebuilds per update, each walking the parent chain.

**Optimization:** Only rebuild breadcrumbs for the updated map and its descendants:
1. Build an index of `parent → children` when maps are added
2. On update of map X, rebuild breadcrumbs for X and recursively for all maps whose parent chain includes X

**Current impact:** At 40-100 maps with updates every 1-60 minutes, the O(n^2) cost is negligible. This is a nice-to-have, not urgent.

---

## Execution order

Recommended sequence for fresh sessions:

1. **Session 1:** Tasks 1.1 + 1.3 + 1.5 (simple fixes, one commit)
2. **Session 2:** Task 1.2 (Go upgrade, may need toolchain install)
3. **Session 3:** Task 2.1 + 2.2 (Dockerfile rewrite + healthcheck)
4. **Session 4:** Task 2.3 (env var config)
5. **Session 5:** Task 1.4 (fetch timeout — needs some design)
6. **Session 6:** Task 3.1 (content tests — biggest effort)
7. **Session 7:** Task 3.2 + 3.3 (integration test + CI)
8. **Sessions 8+:** Phase 4 tasks as desired
