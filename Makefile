BINARY := bin/tf-move-gen
CMD_DIR := ./cmd/tf-move-gen

.PHONY: all build test clean install vet fmt

all: test build

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BINARY) $(CMD_DIR)

test:
	go test -v -race ./...

vet:
	go vet ./...

fmt:
	go fmt ./...

clean:
	rm -rf bin/

install:
	CGO_ENABLED=0 go install $(CMD_DIR)
