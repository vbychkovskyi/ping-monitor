# syntax=docker/dockerfile:1

# ---------- build stage ----------
# Runs on the build host's architecture and cross-compiles, so multi-arch builds don't need emulation.
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder
ARG TARGETOS TARGETARCH

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /ping-monitor .


# ---------- runtime stage ----------
# The static binary needs nothing else: ping uses an ICMP datagram socket, and no TLS is involved.
FROM scratch

COPY --from=builder /ping-monitor /ping-monitor

# nobody:nogroup
USER 65534:65534

EXPOSE 8080

ENTRYPOINT ["/ping-monitor"]
