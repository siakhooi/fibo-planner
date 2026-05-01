#!/bin/bash

set -euxo pipefail

# shellcheck disable=SC1091
. ./release-build.env
# shellcheck disable=SC1091
. ./release.env

(
	docker build . -f docker/Dockerfile \
		-t "$DOCKER_IMAGE_NAME:latest" \
		-t "$DOCKER_IMAGE_NAME:$DOCKER_VERSION"
) 2>&1 | tee docker-build.log
