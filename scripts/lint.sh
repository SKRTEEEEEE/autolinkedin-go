#!/bin/bash
# Run linters for code quality checks

echo "Running golangci-lint..."

# Check if golangci-lint is installed
if ! command -v golangci-lint &> /dev/null; then
    echo "golangci-lint not found. Installing..."
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2
fi

# Run golangci-lint
cd src
golangci-lint run ./...

LINT_EXIT_CODE=$?

if [ $LINT_EXIT_CODE -eq 0 ]; then
    echo "✅ Linting passed!"
else
    echo "❌ Linting failed with exit code $LINT_EXIT_CODE"
fi

exit $LINT_EXIT_CODE
