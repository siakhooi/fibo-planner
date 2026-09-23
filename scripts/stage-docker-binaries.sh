#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "$0")/.."

stage() {
	local os="$1"
	local arch="$2"
	local dest="dist/docker/${os}/${arch}/fibo-planner"
	local matches=(dist/fibo-planner_"${os}"_"${arch}"*/fibo-planner)

	if [[ ${#matches[@]} -ne 1 || ! -f "${matches[0]}" ]]; then
		echo "expected one GoReleaser binary for ${os}/${arch}, found: ${matches[*]:-none}" >&2
		exit 1
	fi

	mkdir -p "$(dirname "$dest")"
	cp "${matches[0]}" "$dest"
	chmod 0755 "$dest"
}

stage linux amd64
stage linux arm64
