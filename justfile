default:
	@just --list
all: build docker-build
build:
	go mod tidy
	gofmt -w -s ./.. 2>&1 | tee gofmt.log
	golangci-lint run 2>&1 | tee golangci-lint.log
	go build -C app -trimpath -ldflags="-s -w -X github.com/siakhooi/fibo-planner/app/versioninfo.Version=$(git describe --tags --always --dirty 2>/dev/null || echo 0.0.0) -X github.com/siakhooi/fibo-planner/app/versioninfo.Commit=$(git rev-parse HEAD 2>/dev/null || echo unknown) -X github.com/siakhooi/fibo-planner/app/versioninfo.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o ../target/server
	./scripts/test.sh
run:
	go run ./app
release:
	scripts/create-release.sh
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
