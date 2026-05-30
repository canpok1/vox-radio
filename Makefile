BINARY_NAME=vox-radio

setup:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0

build:
	go build -o $(BINARY_NAME) ./cmd/vox-radio

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
