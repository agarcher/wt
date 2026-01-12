# Contributing to wt

Thanks for your interest in contributing to `wt`!

## Getting Started

### Prerequisites

- Go 1.21+
- Git 2.5+ (2.13+ recommended)
- Make

### Building

```bash
# Fork and clone the repository
git clone https://github.com/YOUR_USERNAME/wt.git
cd wt

# Build for current platform
make build

# Run tests
make test

# Run linter
make lint
```

### Testing Locally

This repository includes a `.wt.yaml` config, so you can test the tool against itself:

```bash
# Build and add to PATH
make build
export PATH="$(pwd)/build:$PATH"

# Set up shell integration
eval "$(./build/wt init zsh)"  # or bash/fish

# Test commands
wt list
wt create test-feature
wt exit
wt delete test-feature
```

## Architecture

For details on the codebase structure and design patterns, see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Git Workflow

- Fork the repository and submit PRs from your fork
- Write clear commit messages
