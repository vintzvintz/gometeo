# Gometeo — Admin Guide

Quick reference for when you SSH into the VPS to fix things.

---

## Architecture at a glance

```
Internet → Traefik (:80/:443) → traefik-lan (Docker network) → gometeo:1051
```

- **Traefik** handles TLS (wildcard `*.vintz.fr` via Scaleway DNS challenge) and reverse-proxies to gometeo.
- **Gometeo** crawls Météo-France and serves weather pages at `gometeo.vintz.fr`.
- They talk over an external Docker network named `traefik-lan`. This network must exist before gometeo starts.

---

## File locations on the VPS

| Path | What it is |
|------|-----------|
| `~/srv/gometeo/` | App repo (cloned from GitHub) |
| `~/srv/gometeo/docker-compose.yml` | Production compose |
| `~/srv/gometeo/docker-compose.dev.yml` | Local dev / quick test |
| `/etc/traefik/traefik.yml` | Traefik static config (or wherever you mounted it) |
| `/etc/traefik/vintz-wildcard.certstore` | ACME cert storage — do not delete |
| `/var/log/traefik/access.log` | HTTP access log |
| `/var/log/traefik/error.log` | Traefik error/startup log |

---

## Daily operations

### Check status

```bash
# Is gometeo up and healthy?
docker compose -f ~/srv/gometeo/docker-compose.yml ps

# Live logs (last 50 lines, then follow)
docker compose -f ~/srv/gometeo/docker-compose.yml logs --tail=50 -f

# Quick health check
curl -s https://gometeo.vintz.fr/healthz
# Expected: "ok"
```

### Deploy an update

```bash
cd ~/srv/gometeo
git pull
docker compose up -d --build
```

The old container keeps serving during the build. `--build` rebuilds the image from scratch; the Go module cache is baked into the image layers so it's reasonably fast.

### Restart without rebuilding

```bash
docker compose -f ~/srv/gometeo/docker-compose.yml restart
```

### Stop / remove

```bash
docker compose -f ~/srv/gometeo/docker-compose.yml down
# Add --rmi local to also remove the built image
```

---

## Traefik

Traefik is a separate service (presumably its own compose stack). Gometeo just registers itself via Docker labels — no traefik config changes needed when updating gometeo.

### Check traefik is running

```bash
docker ps | grep traefik
curl -s http://localhost:8080/api/rawdata | jq '.routers | keys'  # if dashboard is exposed locally
```

### Certificate renewal

Certificates renew automatically via ACME DNS challenge (Scaleway). The cert store is at `/etc/traefik/vintz-wildcard.certstore`.

If cert renewal fails, check:
1. Scaleway API credentials are set in traefik's environment (env vars `SCALEWAY_*`)
2. `/etc/traefik/` is writable by the traefik container
3. Traefik logs: `/var/log/traefik/error.log`

### traefik-lan network

This external network must exist. Create it once if missing:

```bash
docker network create traefik-lan
```

---

## Troubleshooting

### Site is down — quick checklist

```bash
# 1. Is the container running?
docker compose -f ~/srv/gometeo/docker-compose.yml ps

# 2. Is it healthy?
docker inspect gometeo-app-1 | jq '.[0].State.Health'

# 3. Can it reach its own healthz?
docker exec gometeo-app-1 wget -qO- http://localhost:1051/healthz

# 4. Is traefik routing it? Check the access log
tail -20 /var/log/traefik/access.log

# 5. Is TLS working?
curl -vI https://gometeo.vintz.fr 2>&1 | grep -E 'SSL|certificate|expire'
```

### Container crash-looping

```bash
docker compose -f ~/srv/gometeo/docker-compose.yml logs --tail=100
```

Look for:
- `dial tcp`: upstream (Météo-France) unreachable — harmless if temporary
- `bind: address already in use`: port 1051 conflict
- `failed to load cache`: corrupt `.gob` cache file — delete it and restart

### Météo-France auth errors

The crawler uses a cookie + ROT13 bearer token. If Météo-France changes their auth, maps will stop updating. Symptom: logs full of `401` or `403` responses. Check `crawl/` package for the auth logic.

### Build fails (Go version)

The Dockerfile pins `golang:1.26.2-bookworm`. If that image tag disappears from Docker Hub, update the `FROM` line in `Dockerfile`.

### "no space left on device"

Old images accumulate. Clean up:

```bash
docker image prune -f        # remove dangling images
docker system prune -f       # broader cleanup (skips volumes)
docker system df             # see what's using space
```

---

## Logs

Gometeo logs to stdout (structured JSON via `slog`). Docker captures them.

```bash
# All logs since last restart
docker compose -f ~/srv/gometeo/docker-compose.yml logs

# Filter for errors only
docker compose -f ~/srv/gometeo/docker-compose.yml logs | grep '"level":"ERROR"'

# Traefik access log — see all requests to gometeo
grep gometeo.vintz.fr /var/log/traefik/access.log | tail -20
```

Log retention: Docker's default is no rotation (grows forever). See note below.

---

## Known quirks and caveats

- **`build.network: host`** in docker-compose.yml — this lets `go mod download` during build use the host network. Required on this VPS if the builder would otherwise lack internet access inside a bridge network. No effect at runtime.
- **Hot/cold update logic** — maps recently viewed update every 1–60 min; idle maps update every 4–5 hours. After a fresh deploy, all maps start cold. Expect ~5 min before popular maps are warm again.
- **`-limit 40`** — the crawler stops after fetching 40 maps. Increase this flag in `docker-compose.yml` if coverage seems thin.
- **No persistent volume** — the in-memory store is rebuilt on each startup by re-crawling. First fetch takes ~1–2 min. The `-cache` flag (`.gob` file) can speed this up but is not enabled in production.

---

## Suggested improvements (not yet done)

- **Log rotation**: add `logging` config to docker-compose.yml to cap log size:
  ```yaml
  logging:
    driver: "json-file"
    options:
      max-size: "10m"
      max-file: "3"
  ```
- **Resource limits**: add `mem_limit: 256m` to prevent runaway memory usage.
- **Traefik security headers**: add a middleware in traefik for HSTS, X-Frame-Options, etc. Not urgent for a personal site.
- **Oneshot cache in production**: mount a volume and use `-cache /data/cache.gob` to survive restarts without a full re-crawl.
