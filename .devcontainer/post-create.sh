#!/bin/bash
set -e

# install Claude Code
curl -fsSL https://claude.ai/install.sh | bash

# install Todoist CLI (td) for task management workflow scripts
npm install -g @doist/todoist-cli

# install vox-actor via Homebrew
eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"
brew tap canpok1/tap
brew install --cask vox-actor

# install build tools (golangci-lint, goreleaser, lefthook) and activate git hooks
make setup
