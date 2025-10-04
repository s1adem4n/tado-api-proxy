#!/bin/bash

export CGO_ENABLED=0
export GOOS=linux

# build for amd64, arm64 and armv7
GOARCH=amd64 go build -o build/main-amd64 ./cmd/main.go
GOARCH=arm64 go build -o build/main-arm64 ./cmd/main.go
GOARCH=arm GOARM=7 go build -o build/main-armv7 ./cmd/main.go