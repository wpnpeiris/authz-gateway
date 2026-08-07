BINARY_NAME := authz-gateway
BUILD_DIR := bin
VERSION ?= $(shell git describe --always --tags 2>/dev/null || echo dev)
COVERPROFILE := coverage.out

.PHONY: all build fmt fmt-fix format vet test coverage coverage-report coverage-html run clean clean-coverage

all: build test

$(BUILD_DIR):
	@mkdir -p $(BUILD_DIR)

build: $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux go build -ldflags="-X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/authz-gateway

coverage:
	go test -race -covermode=atomic -coverprofile=$(COVERPROFILE) ./...

coverage-report: coverage
	@go tool cover -func=$(COVERPROFILE) | tail -n 1

coverage-html: coverage
	@go tool cover -html=$(COVERPROFILE) -o coverage.html

fmt:
	@files=$$(gofmt -s -l .); if [ -n "$$files" ]; then \
		echo "The following files are not gofmt-ed:"; echo "$$files"; exit 1; \
	else echo "gofmt check passed"; fi

format fmt-fix:
	gofmt -s -w .

vet:
	go vet ./...

test: build fmt vet
	go test -race -count=1 ./...

run: build
	$(BUILD_DIR)/$(BINARY_NAME) --listen 0.0.0.0:8080

clean:
	rm -rf $(BUILD_DIR)

clean-coverage:
	rm -f $(COVERPROFILE) coverage.html
