# syntax=docker/dockerfile:1

FROM golang:1.20-alpine AS builder

# Install gcc for cgo (required by go-sqlite3)
RUN apk add build-base

WORKDIR /build

COPY . /build

RUN go mod download
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o score

FROM alpine

ENV SCORE_LISTEN=0.0.0.0:8080

WORKDIR /app

COPY --from=builder /build/score /app/score

CMD ./score $SCORE_LISTEN
