.PHONY: build test

build:
	go build -o app ./main.go

test:
	go test ./...