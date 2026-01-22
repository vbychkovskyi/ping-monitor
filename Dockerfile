# ---------- build stage ----------
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Required for ping at runtime
RUN apk add --no-cache ca-certificates

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o uptime-monitor


# ---------- runtime stage ----------
FROM alpine:3.19

# ping + certs
RUN apk add --no-cache iputils ca-certificates

WORKDIR /app
COPY --from=builder /app/uptime-monitor .

# Non-root user
RUN adduser -D appuser
USER appuser

EXPOSE 8080

CMD ["./uptime-monitor"]
