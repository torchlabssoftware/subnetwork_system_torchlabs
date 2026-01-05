# -------------------------------
# 1. Build Stage
# -------------------------------
FROM golang:1.25 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# -------------------------------
# 2. Run Stage (lightweight)
# -------------------------------
FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/server .

EXPOSE 8080

CMD ["./server"]
