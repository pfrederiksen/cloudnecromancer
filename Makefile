.PHONY: build test lint release snapshot

build:
	go build -o bin/cloudnecromancer ./main.go

test:
	go test ./...

lint:
	golangci-lint run

release:
	goreleaser release --clean

snapshot:
	goreleaser release --snapshot --clean
