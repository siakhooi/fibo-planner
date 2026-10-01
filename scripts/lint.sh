#!/usr/bin/env bash

set -euo pipefail

golangci-lint run --timeout 5m
