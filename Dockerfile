# syntax=docker/dockerfile:1

ARG GO_VERSION=1.26
FROM golang:${GO_VERSION}-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/naeos-intelligence ./cmd/naeos-intelligence
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/naeos-intelligence-migrate ./cmd/naeos-intelligence-migrate

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S naeos && adduser -S naeos -G naeos

WORKDIR /app

COPY --from=builder /bin/naeos-intelligence /bin/naeos-intelligence
COPY --from=builder /bin/naeos-intelligence-migrate /bin/naeos-intelligence-migrate
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

USER naeos

ENV NAEOS_INTELLIGENCE_HTTP_ADDR=0.0.0.0:8080 \
    NAEOS_INTELLIGENCE_APP_ENVIRONMENT=production \
    NAEOS_INTELLIGENCE_APP_LOG_LEVEL=info

EXPOSE 8080

ENTRYPOINT ["/bin/naeos-intelligence"]
