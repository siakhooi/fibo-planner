set shell := ["bash", "-cuo", "pipefail"]
default:
	@just --list

all: check docker-build build-release

check: fmt-check lint test build

fmt-check:
	./scripts/fmt-check.sh

lint:
	./scripts/lint.sh

test:
	./scripts/test.sh

build:
	./scripts/build.sh

tidy:
	go mod tidy

build-release:
	go tool goreleaser release --snapshot --clean --skip=publish

run *args:
	go run ./app {{args}}

release:
	./scripts/create-release.sh

clean:
	rm -rf *.log target test-* dist

docker-build:
	./scripts/docker-build.sh

docker-run:
	docker run -p 8080:8080 siakhooi/fibo-planner

curl-ws:
	curl -sS -N  ws://localhost:8080/ws

websocat-ws:
	websocat ws://localhost:8080/ws
