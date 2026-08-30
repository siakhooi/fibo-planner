run:
	go run ./app

build:
	go mod tidy
	gofmt -w -s ./.. 2>&1 | tee gofmt.log
	golangci-lint run 2>&1 | tee golangci-lint.log
	go build -C app -o ../target/server
	./scripts/test.sh

all: build docker-build

release:
	scripts/create-release.sh

clean:
	rm -f *.log

docker-build:
	docker build -t siakhooi/fibo-planner -f docker/Dockerfile .

docker-run:
	docker run -p 8080:8080 siakhooi/fibo-planner

curl-ws:
	curl -sS -N  ws://localhost:8080/ws

websocat-ws:
	websocat ws://localhost:8080/ws
