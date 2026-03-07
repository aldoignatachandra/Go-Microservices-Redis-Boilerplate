#!/bin/bash
#
# Install Git hooks for Go Microservices Redis PubSub Boilerplate
# Run this script once after cloning the repository
#

set -e

echo "🔧 Installing Git hooks..."

# Make hook scripts executable
chmod +x .githooks/pre-commit
chmod +x .githooks/commit-msg

# Configure Git to use the .githooks directory
git config core.hooksPath .githooks

echo "✅ Git hooks installed successfully!"
echo ""
echo "📋 Installed hooks:"
echo "   • pre-commit  - Runs gofmt, go vet, golangci-lint"
echo "   • commit-msg  - Validates commit message format"
echo ""
echo "📝 Commit message format: <type>: <description>"
echo "   Allowed types: add, update, fix, feat, refactor, docs, test, chore"
echo ""
