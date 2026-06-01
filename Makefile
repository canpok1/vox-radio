BINARY_NAME=vox-radio
VERSION ?= dev
LDFLAGS=-X github.com/canpok1/vox-radio/internal/cli.version=$(VERSION)
PROFILE ?= sample-profiles/tech_profile.yaml
OUT_DIR ?= output/$(shell date +%Y%m%d%H%M%S)

setup:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) ./cmd/vox-radio

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
	go install -ldflags "$(LDFLAGS)" ./cmd/vox-radio

docs:
	go run ./tools/gendocs

run-sample: build
	./$(BINARY_NAME) run --profile "$(PROFILE)" --out-dir "$(OUT_DIR)"

check-samples: build
	./$(BINARY_NAME) config check vox-radio.yaml
	./$(BINARY_NAME) config check internal/cli/templates/vox-radio.yaml
	cd internal/cli/templates && "$(CURDIR)/$(BINARY_NAME)" profile check profile.yaml
	for f in sample-profiles/*.yaml; do ./$(BINARY_NAME) profile check "$$f"; done

all: build

.PHONY: all setup build clean test fmt lint install docs check-samples run-sample
