.PHONY: build test lint fmt coverage clean

build:
	go build -o bin/kubecfg .

test:
	go test -v -race -coverprofile=coverage.out ./...

coverage: test
	go tool cover -html=coverage.out

lint:
	golangci-lint run --timeout=5m

fmt:
	gofmt -w -s .
	go mod tidy

clean:
	rm -rf bin/
	rm -f coverage.out

.DEFAULT_GOAL := build
