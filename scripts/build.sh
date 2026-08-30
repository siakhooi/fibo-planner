#!/usr/bin/env bash

set -xe
gofmt -d ./..
golangci-lint run
