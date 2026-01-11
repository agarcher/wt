#!/bin/bash
set -euo pipefail

# Generate Homebrew formula from template
# Usage: generate-formula.sh [output-file]
#
# Required environment variables:
#   VERSION           - Version number (without 'v' prefix)
#   SHA_DARWIN_AMD64  - SHA256 for darwin-amd64 tarball
#   SHA_DARWIN_ARM64  - SHA256 for darwin-arm64 tarball
#   SHA_LINUX_AMD64   - SHA256 for linux-amd64 tarball
#   SHA_LINUX_ARM64   - SHA256 for linux-arm64 tarball

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEMPLATE="${SCRIPT_DIR}/homebrew-formula.tmpl"

# Validate required variables
for var in VERSION SHA_DARWIN_AMD64 SHA_DARWIN_ARM64 SHA_LINUX_AMD64 SHA_LINUX_ARM64; do
    if [[ -z "${!var:-}" ]]; then
        echo "Error: $var is required" >&2
        exit 1
    fi
done

# Generate formula by substituting placeholders
sed -e "s/{{VERSION}}/${VERSION}/g" \
    -e "s/{{SHA_DARWIN_AMD64}}/${SHA_DARWIN_AMD64}/g" \
    -e "s/{{SHA_DARWIN_ARM64}}/${SHA_DARWIN_ARM64}/g" \
    -e "s/{{SHA_LINUX_AMD64}}/${SHA_LINUX_AMD64}/g" \
    -e "s/{{SHA_LINUX_ARM64}}/${SHA_LINUX_ARM64}/g" \
    "${TEMPLATE}"
