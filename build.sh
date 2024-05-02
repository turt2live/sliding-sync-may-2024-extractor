#! /bin/bash
set -x
GOOS=windows GOARCH=amd64 go build -v -o bin/sliding-sync-may-2024-extractor-windows-amd64
GOOS=linux GOARCH=amd64 go build -v -o bin/sliding-sync-may-2024-extractor-linux-amd64
GOOS=darwin GOARCH=amd64 go build -v -o bin/sliding-sync-may-2024-extractor-darwin-amd64
