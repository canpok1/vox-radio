BINARY_NAME=vox-radio
VERSION ?= dev

setup:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0

build:
	go build -ldflags "-X github.com/canpok1/vox-radio/internal/cli.version=$(VERSION)" -o $(BINARY_NAME) ./cmd/vox-radio

clean:
	go clean
	rm -f $(BINARY_NAME)

test:
	go test ./...

fmt:
	go fmt ./...

lint:
	golangci-lint run ./...

install:
	go install ./cmd/vox-radio

docs:
	go run ./tools/gendocs

all: build

.PHONY: all setup build clean test fmt lint install docs
