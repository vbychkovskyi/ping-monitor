# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A single-file Go HTTP service (`main.go`, stdlib only, no dependencies) that reports whether a host is online. Its real purpose is detecting mains electricity at home: an external uptime checker calls it for `192.168.55.39`, a mains-only Tuya Wi-Fi warm-floor switcher that goes offline in a power cut. Don't use `192.168.55.87` — it can run on a generator and stay online without mains power.

- `GET /health?ip=<host>` → `200 {"status":"UP"}`, `503 {"status":"DOWN"}`, `400` if `ip` is missing.
- Listens on `:8080` (hardcoded).
- Each attempt (`probe`) runs ICMP ping and a TCP connect to port 6668 (Tuya local protocol) concurrently under a 1s context timeout; either succeeding means UP. `isUp` retries up to 3 times with 300ms gaps, so DOWN takes ~3.6s. Timing constants are at the top of `main.go`.
- Ping shells out to the system `ping` binary rather than using raw sockets, so the runtime image needs `iputils` and the process can run as a non-root user. It has no `-W`: the context kills it instead, because `-W` is seconds on Linux but milliseconds on macOS.

## Commands

```sh
go run .                       # run locally on :8080
go build -o ping-monitor .     # local binary (the untracked ./ping-monitor is a stale build artifact)
go vet ./...
curl 'localhost:8080/health?ip=192.168.55.39'

docker compose up --build      # dev: builds from Dockerfile, exposed on host port 8088
docker compose -f prod/docker-compose.yml up -d   # prod: pulls ghcr.io/vbychkovskyi/ping-monitor:latest, host port 8089
```

There are no tests yet.

## Gotchas

- Tuya devices allow only a few concurrent local connections; frequent polling or holding connections open can block other integrations.
- The Go toolchain may not be on `PATH`; it is installed at `~/sdk/go1.25.6/bin`.
- Naming is inconsistent: the Dockerfile builds the binary as `uptime-monitor` and the dev compose service/container is `uptime-monitor`, while the module, prod service, and published image are `ping-monitor`.
- Nothing in this repo builds or pushes the ghcr.io image; that happens outside the repo.
