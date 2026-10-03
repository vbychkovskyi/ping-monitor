# ping-monitor

A small HTTP service that reports whether a host answers ICMP ping. Point an external uptime checker at it to monitor hosts that only respond to ping.

## API

```
GET /health?ip=<host>
```

| Result | Status | Body |
|---|---|---|
| Host answered one ping | `200` | `{"status":"UP"}` |
| No reply, or no answer within 2s | `503` | `{"status":"DOWN"}` |
| `ip` parameter missing | `400` | plain-text error |

`ip` can be an IP address or a hostname. The service listens on port `8080`.

```sh
curl 'localhost:8080/health?ip=1.1.1.1'
```

## Running

Locally (requires Go 1.25):

```sh
go run .
```

With Docker, building from source (exposed on host port `8088`):

```sh
docker compose up --build
```

In production, using the published image `ghcr.io/vbychkovskyi/ping-monitor:latest` (exposed on host port `8089`):

```sh
docker compose -f prod/docker-compose.yml up -d
```

## How it works

The service calls the system `ping` (`ping -c 1 -W 1 <host>`) instead of opening raw sockets, so it runs as a non-root user. The container image installs `iputils` for this. On macOS `-W` is in milliseconds rather than seconds, so local runs time out each ping much sooner than the container does.
