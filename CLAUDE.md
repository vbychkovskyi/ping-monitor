# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A single-file Go HTTP service (`main.go`, stdlib only, no dependencies) that reports whether a host is online. Its real purpose is detecting mains electricity at home: an external uptime checker calls it for `192.168.55.39`, a mains-only Tuya Wi-Fi warm-floor switcher that goes offline in a power cut. Don't use `192.168.55.87` — it can run on a generator and stay online without mains power.

- `GET /health?ip=<host>` → `200 {"status":"UP"}`, `503 {"status":"DOWN"}`, `400` if `ip` is missing.
- Listens on `:8080` (hardcoded).
- Each attempt (`probe`) runs ICMP ping and a TCP connect to port 6668 (Tuya local protocol) concurrently under a 1s context timeout; either succeeding means UP. `isUp` retries up to 3 times with 300ms gaps, so DOWN takes ~3.6s. Timing constants are at the top of `main.go`.
- Ping is implemented in Go with `golang.org/x/net/icmp` over an unprivileged `udp4` ICMP socket (no root, no `ping` binary). On Linux that depends on `net.ipv4.ping_group_range`, which Docker opens to all groups except under `network_mode: host`. If ping can't open its socket it logs and returns false; the TCP check still works.
- The image is `FROM scratch` with only the static binary, running as `65534:65534`. Anything added at runtime (shell, curl, CA certs for TLS) must be copied in explicitly; there is no package manager. `.dockerignore` is an allowlist: only `go.mod`, `go.sum` and `*.go` reach the build context.

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
- The Go toolchain may not be on `PATH`; it is installed at `~/sdk/go1.25.6/bin`. Keep `go.mod` at Go 1.25 to match the `golang:1.25` builder: newer `golang.org/x/net` releases (v0.59+) require Go 1.26, so use `GOTOOLCHAIN=local` when running `go get` to stop it bumping the `go` line.
- The dev compose service/container is still named `uptime-monitor`; everything else (module, binary, prod service, published image) is `ping-monitor`.
- Nothing in this repo builds or pushes the ghcr.io image; that happens outside the repo.
