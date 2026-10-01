#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "$0")/.."

mode="${1:-snapshot}"
args=(build --clean)
case "$mode" in
snapshot)
	args+=(--snapshot)
	;;
release) ;;
*)
	echo "usage: scripts/build-docker-binaries.sh [snapshot|release]" >&2
	exit 1
	;;
esac

go tool -modfile=tools/go.mod goreleaser "${args[@]}"
./scripts/stage-docker-binaries.sh
