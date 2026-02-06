# -------------------------------
# 1. Build Stage
# -------------------------------
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o proxy .

# -------------------------------
# 2. Run Stage (minimal scratch image)
# -------------------------------
FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
COPY --from=builder /app/proxy /proxy

ENTRYPOINT ["/proxy"]
CMD ["--version"]
