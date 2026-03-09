.PHONY: test check tidy build

test:
	go test ./...

check:
	go vet ./...

tidy:
	go mod tidy

build:
	go build ./...
