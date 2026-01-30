#!/bin/bash

set -e  # Exit on error

cd web
bun install --frozen-lockfile
bun run build
cd ..

export CGO_ENABLED=0
export GOOS=linux

go mod download

GOARCH=amd64 go build -o build/main-amd64 ./cmd/main.go &
pid1=$!
GOARCH=arm64 go build -o build/main-arm64 ./cmd/main.go &
pid2=$!

# Wait for all background jobs and capture their exit codes
wait $pid1 || exit 1
wait $pid2 || exit 1