BINARY_NAME=tf-move-gen
BIN_DIR=bin
CMD_DIR=./cmd/tf-move-gen
GO=go

.PHONY: all build test test-coverage lint clean install docs-serve docs-update

all: build

build:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags="-s -w" -o $(BIN_DIR)/$(BINARY_NAME) $(CMD_DIR)

test:
	$(GO) test -v -race -timeout 60s ./...

test-coverage:
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

lint:
	$(GO) vet ./...
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipped extended linting"; \
	fi

clean:
	rm -rf $(BIN_DIR)
	rm -f coverage.out

install:
	CGO_ENABLED=0 $(GO) install -trimpath -ldflags="-s -w" $(CMD_DIR)

docs-serve:
	@echo "Serving documentation at http://localhost:8000 ..."
	python3 -m http.server 8000 --directory docs

docs-update:
	curl -sSL https://raw.githubusercontent.com/smford/md-web-renderer/main/index.html -o docs/index.html
	touch docs/.nojekyll
