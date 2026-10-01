#!/usr/bin/env bash

set -euo pipefail

version=$(git describe --tags --always --dirty 2>/dev/null || echo 0.0.0)
commit=$(git rev-parse HEAD 2>/dev/null || echo unknown)
date=$(date -u +%Y-%m-%dT%H:%M:%SZ)

mkdir -p target
go build -C app -trimpath \
	-ldflags="-s -w -X github.com/siakhooi/fibo-planner/app/versioninfo.Version=${version} -X github.com/siakhooi/fibo-planner/app/versioninfo.Commit=${commit} -X github.com/siakhooi/fibo-planner/app/versioninfo.Date=${date}" \
	-o ../target/server
