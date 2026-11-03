.PHONY: fmt vet test build check

fmt:
	gofmt -w ./cmd

vet:
	go vet ./...

test:
	go test ./...

build:
	go build ./...

check: vet test build
