# ping-monitor

A small HTTP service that reports whether a host on the local network is online. It is used to detect whether mains electricity is present at home: it checks a mains-powered Tuya Wi-Fi warm-floor switcher (`192.168.55.39`, static DHCP lease), which drops off the network when the power goes out. Point an external uptime checker at it.

## API

```
GET /health?ip=<host>
```

| Result | Status | Body |
|---|---|---|
| Host answered ping or accepted a TCP connection | `200` | `{"status":"UP"}` |
| No answer after 3 attempts (~3.6s) | `503` | `{"status":"DOWN"}` |
| `ip` parameter missing | `400` | plain-text error |

`ip` can be an IP address or a hostname. The service listens on port `8080`.

```sh
curl 'localhost:8080/health?ip=192.168.55.39'
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

The image is built `FROM scratch` and contains only the static binary (~5.6 MB), running as `nobody`.

In production, using the published image `ghcr.io/vbychkovskyi/ping-monitor:latest` (exposed on host port `8089`):

```sh
docker compose -f prod/docker-compose.yml up -d
```

## How it works

Each attempt runs two checks at the same time, with a 1s timeout:

- ICMP ping, sent from Go over an unprivileged ICMP datagram socket, so it needs neither root nor a `ping` binary. On Linux the process's group must be allowed by `net.ipv4.ping_group_range`; Docker allows all groups by default (but not with `network_mode: host`, where the host's setting applies).
- A TCP connection to port `6668`, the Tuya local-protocol port. The connection is closed immediately; no login is needed.

The host is UP as soon as either check succeeds. If both fail, the service waits 300ms and tries again, up to 3 attempts, so a single delayed Wi-Fi reply doesn't cause a false DOWN. Tuya devices accept only a few local connections at once, so don't poll more often than every 30–60s.

Use `192.168.55.39` only. The other switcher (`192.168.55.87`) is sometimes powered by a generator and can stay online during a power cut.
