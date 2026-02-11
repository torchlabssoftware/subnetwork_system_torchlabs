# -------------------------------
# 1. Build Stage
# -------------------------------
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o proxy .

# -------------------------------
# 2. Run Stage (Alpine for shell/openssl support)
# -------------------------------
FROM alpine:3.19

# Install runtime dependencies (openssl for keygen, ca-certificates)
RUN apk add --no-cache ca-certificates tzdata openssl

ENV TZ=UTC
RUN cp /usr/share/zoneinfo/${TZ} /etc/localtime && echo "${TZ}" > /etc/timezone
COPY --from=builder /app/proxy /proxy
COPY entrypoint.sh /entrypoint.sh

RUN sed -i 's/\r$//' /entrypoint.sh && chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
CMD ["/proxy", "--version"]
