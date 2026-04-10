# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Gometeo** is a Go weather forecasting web application that crawls Météo-France, processes multi-source forecast data, and serves it as HTML pages + JSON API with a Vue.js frontend.

## Commands

```bash
# Build
go build -o gometeo

# Run (dev mode: fetch 5 maps once, then serve)
./gometeo -oneshot -limit 5 -fastupdate

# Run (production-like)
./gometeo -limit 40

# Tests
go test ./...
go test -v ./crawl/...    # single package
go test -run TestName ./mfmap/...  # single test

# Update test fixtures from upstream
cd test_data && ./update.sh

# Code quality
go fmt ./...
go vet ./...

# Docker
docker-compose up
```

## CLI Flags

| Flag | Default | Purpose |
|------|---------|---------|
| `-addr` | `:1051` | Server listen address |
| `-limit` | `0` (unlimited) | Stop crawling after N maps |
| `-oneshot` | `false` | Fetch once, then serve (dev/debug) |
| `-vue` | `prod` | Vue.js build: `dev` or `prod` |
| `-fastupdate` | `false` | Reduce update intervals (dev: 30min→1min) |

## Architecture

### High-Level Data Flow

```
Météo-France (upstream)
    ↓
[crawl] — HTTP client with auth (mfsession cookie, ROT13 Bearer token)
    ↓  channels (chMap, chPicto)
[content] — thread-safe in-memory store, rebuilds http.ServeMux on each update
    ↓
[mfmap/handlers] — HTTP handlers: HTML page, JSON data, SVG map, picto icons
    ↓
Browser (Vue.js SPA)
```

### Package Responsibilities

- **`appconf`** — CLI flags, constants, upstream URL, cache ID (8-char hex for cache-busting), hot/cold update intervals
- **`crawl`** — Multi-level tree crawl of weather map hierarchy. `crawl.Start()` runs `ModeOnce` or `ModeForever`. Uses cache policies (`CacheDefault`, `CacheUpdate`, `CacheDisabled`, `CacheOnly`)
- **`mfmap`** — Core `MfMap` type: stores parsed forecasts, geography (GeoJSON), SVG map, pictos, parent chain (breadcrumbs), and atomic hit/update stats. `IsHot()` / `DurationToUpdate()` drive refresh scheduling
- **`geojson`** — `MultiforecastData`, `Forecast`, `Daily` types; custom JSON unmarshalling for upstream quirks; `Merge()` preserves historical forecast data across updates
- **`content`** — `Meteo` struct: fan-in from crawler channels via `Receive()`, mutex-protected `mapStore`/`pictoStore`, hot-swappable `meteoMux`. `Updatable()` selects the next map needing refresh
- **`mfmap/handlers`** — Registers URL patterns: `/{path}` (HTML), `/{path}/data` (JSON), `/{path}/{cacheid}/svg` (SVG). The `/data` endpoint marks the map as "hit" (making it "hot")
- **`server`** — Entry point; middleware chain: logging → legacy `.html` redirect → static files → `Meteo.ServeHTTP()`
- **`static`** — Embedded static assets (JS, CSS, fonts, favicon) via `go:embed`; served with immutable cache headers using `CacheId()` in URLs
- **`svgtools`** — Crops SVG maps using `etree`
- **`stringfloat`** — Custom JSON unmarshaller for lat/lng fields that upstream sends as either strings or floats

### Hot/Cold Update Logic

Maps that have been recently viewed (`IsHot()`) update every 1–60 minutes. Idle maps update every 4–5 hours. This avoids hammering upstream for unused regions.

### Concurrency Model

- Crawler and server run in separate goroutines connected by channels
- `mapStore` and `pictoStore` use mutexes; the `http.ServeMux` is rebuilt atomically on each map update
- Breadcrumb chains are rebuilt for **all** maps on every update (O(n²) but acceptable at typical map counts of 40–100)
- Atomic fields (`lastHit`, `lastUpdate`, `hitCount`) on `MfMap` need no locks

### Test Fixtures

- `test_data/` contains HTML pages and JSON responses from upstream
- `testutils/` provides mock HTTP handlers and map helpers reused across packages
