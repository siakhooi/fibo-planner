#!/usr/bin/env bash

set -euo pipefail

diff=$(gofmt -d -s .)
if [[ -n "$diff" ]]; then
	printf '%s\n' "$diff"
	echo "gofmt: files differ; run gofmt -w -s ." >&2
	exit 1
fi
