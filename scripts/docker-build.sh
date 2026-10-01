#!/usr/bin/env bash

set -euxo pipefail

# shellcheck disable=SC1091
. ./release-build.env

(
	./scripts/build-docker-binaries.sh snapshot

	docker build dist/docker -f docker/Dockerfile \
		-t "$DOCKER_IMAGE_NAME:latest" \
		-t "$DOCKER_IMAGE_NAME:$VERSION"
) 2>&1 | tee docker-build.log
