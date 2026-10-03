# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A single-file Go HTTP service (`main.go`, stdlib only, no dependencies) that reports whether a host answers ICMP ping. Intended as a target for external uptime checkers.

- `GET /health?ip=<host>` → `200 {"status":"UP"}` if one ping succeeds, `503 {"status":"DOWN"}` on failure or after a 2s handler timeout, `400` if `ip` is missing.
- Listens on `:8080` (hardcoded).
- Pinging shells out to the system `ping` binary (`ping -c 1 -W 1 <ip>`) rather than using raw sockets, so the runtime image needs `iputils` and the process can run as a non-root user.

## Commands

```sh
go run .                       # run locally on :8080
go build -o ping-monitor .     # local binary (the untracked ./ping-monitor is a stale build artifact)
go vet ./...
curl 'localhost:8080/health?ip=1.1.1.1'

docker compose up --build      # dev: builds from Dockerfile, exposed on host port 8088
docker compose -f prod/docker-compose.yml up -d   # prod: pulls ghcr.io/vbychkovskyi/ping-monitor:latest, host port 8089
```

There are no tests yet.

## Gotchas

- `ping -W` means seconds on Linux (iputils, used in the container) but milliseconds on macOS, so local `go run` on a Mac uses a much shorter per-ping timeout than production. Verify ping behavior in the container.
- Naming is inconsistent: the Dockerfile builds the binary as `uptime-monitor` and the dev compose service/container is `uptime-monitor`, while the module, prod service, and published image are `ping-monitor`.
- Nothing in this repo builds or pushes the ghcr.io image; that happens outside the repo.
