#!/bin/bash
set -e

# install Claude Code
curl -fsSL https://claude.ai/install.sh | bash

# install vox-actor via Homebrew
eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"
brew tap canpok1/tap
brew install --cask vox-actor
