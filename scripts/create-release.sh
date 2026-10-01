#!/usr/bin/env bash

set -euo pipefail

if [[ $# -ne 0 ]]; then
	echo "just release takes no arguments; edit release.env, then run just release" >&2
	exit 1
fi

# shellcheck disable=SC1091
. ./release-build.env
# shellcheck disable=SC1091
. ./release.env

if [[ -z "${VERSION}" ]]; then
	echo "VERSION is empty in release-build.env" >&2
	exit 1
fi
if [[ -z "${RELEASE_TITLE}" ]]; then
	echo "RELEASE_TITLE is empty in release.env" >&2
	exit 1
fi
if [[ -z "${RELEASE_NOTE:-}" ]]; then
	echo "RELEASE_NOTE is empty in release.env" >&2
	exit 1
fi

gh release create "v${VERSION}" --title "${RELEASE_TITLE}" --notes "${RELEASE_NOTE}" --latest
