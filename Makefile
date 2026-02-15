.PHONY: test test-cover test-race lint build clean

test:
	go test ./... -v -count=1

test-cover:
	go test ./... -coverprofile=cover.out -covermode=atomic
	go tool cover -html=cover.out -o cover.html

test-race:
	go test ./... -race -count=1

test-integration:
	go test -tags=integration ./... -v -count=1

lint:
	golangci-lint run ./...

build:
	go build -o git-wt .
