#!/usr/bin/env bash

set -euo pipefail
set -x

go test -v -json -covermode=atomic -coverpkg=./... -coverprofile=test-coverage.out ./... | tee test-report.json

go tool cover -func=test-coverage.out
go tool cover -html=test-coverage.out -o test-coverage.html
