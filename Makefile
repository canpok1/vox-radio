BINARY_NAME=vox-radio
VERSION ?= dev
LDFLAGS=-X github.com/canpok1/vox-radio/internal/cli.version=$(VERSION)
PROFILE ?= sample/episode-spec.yaml
OUT_DIR ?= output/$(shell date +%Y%m%d%H%M%S)

setup:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0
	go install github.com/goreleaser/goreleaser/v2@v2.14.3

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
	./$(BINARY_NAME) init --sample --output-dir sample
	./$(BINARY_NAME) --config sample/vox-radio.yaml episodegen --spec "$(PROFILE)" --out-dir "$(OUT_DIR)" --log-dir "$(OUT_DIR)"

check-samples: build
	./$(BINARY_NAME) --config internal/cli/templates/vox-radio.yaml config check
	cd internal/cli/templates && "$(CURDIR)/$(BINARY_NAME)" episodegen check episode-spec.yaml
	cd internal/cli/templates && "$(CURDIR)/$(BINARY_NAME)" assets check assets/assets.yaml
	./$(BINARY_NAME) init --sample --output-dir sample
	./$(BINARY_NAME) --config sample/vox-radio.yaml config check
	./$(BINARY_NAME) --config sample/vox-radio.yaml episodegen check sample/episode-spec.yaml
	./$(BINARY_NAME) assets check sample/assets/assets.yaml
	./$(BINARY_NAME) feedgen check sample/feed-spec.yaml
	./$(BINARY_NAME) slackpost check sample/slack-spec.yaml

release-check:
	goreleaser check

eval:
	go test -tags=eval -count=1 -v -timeout 30m ./internal/eval/...

e2e:
	go test -tags=e2e -count=1 -v -timeout 10m ./e2e/...

all: build

.PHONY: all setup build clean test fmt lint install docs check-samples run-sample release-check eval e2e
