#!/usr/bin/env bash

set -euxo pipefail

# shellcheck disable=SC1091
. ./release-build.env

(
	go tool -modfile=tools/go.mod goreleaser build --snapshot --clean
	./scripts/stage-docker-binaries.sh

	docker build dist/docker -f docker/Dockerfile \
		-t "$DOCKER_IMAGE_NAME:latest" \
		-t "$DOCKER_IMAGE_NAME:$VERSION"
) 2>&1 | tee docker-build.log
