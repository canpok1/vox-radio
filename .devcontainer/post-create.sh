#!/bin/bash
set -e

# install Claude Code
curl -fsSL https://claude.ai/install.sh | bash

# install build tools (golangci-lint, lefthook) and activate git hooks
make setup
