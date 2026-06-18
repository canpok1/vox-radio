BINARY_NAME=vox-radio
VERSION ?= dev
LDFLAGS=-X github.com/canpok1/vox-radio/internal/cli.version=$(VERSION)
PROFILE ?= sample/episode-spec.yaml
OUT_DIR ?= output/$(shell date +%Y%m%d%H%M%S)

# 開発ツールのバージョン
GOLANGCI_LINT_VERSION ?= v2.12.2
GORELEASER_VERSION ?= v2.14.3
LEFTHOOK_VERSION ?= v2.1.8
GOBIN ?= $(shell go env GOPATH)/bin

setup:
	# go install（ソースからビルド）は遅いため、ビルド済みバイナリを取得する。
	# golangci-lint: 公式 install.sh がチェックサム検証込みでバイナリを配置する。
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(GOBIN) $(GOLANGCI_LINT_VERSION)
	# lefthook: GitHub Releases のバイナリを取得する（goreleaser は release-check で都度取得）。
	./scripts/install-lefthook.sh $(LEFTHOOK_VERSION) $(GOBIN)
	$(GOBIN)/lefthook install

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
	./$(BINARY_NAME) assets check sample-assets/assets.yaml

release-check:
	# goreleaser を setup でインストールせず、公式 run スクリプトで都度取得して実行する。
	curl -sfL https://goreleaser.com/static/run | VERSION=$(GORELEASER_VERSION) bash -s -- check

eval:
	go test -tags=eval -count=1 -v -timeout 30m ./internal/eval/...

e2e:
	go test -tags=e2e -count=1 -v -timeout 10m ./e2e/...

all: build

.PHONY: all setup build clean test fmt lint install docs check-samples run-sample release-check eval e2e
