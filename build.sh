#!/bin/bash

export CGO_ENABLED=0
export GOOS=linux

go mod download

GOARCH=amd64 go build -o build/main-amd64 ./cmd/main.go &
GOARCH=arm64 go build -o build/main-arm64 ./cmd/main.go &
GOARCH=arm GOARM=7 go build -o build/main-arm ./cmd/main.go &

wait