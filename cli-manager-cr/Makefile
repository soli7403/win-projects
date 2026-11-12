.PHONY: build test vet tidy

build:
	go build ./...

test:
	go test -race ./...

vet:
	go vet ./...

tidy:
	go mod tidy
