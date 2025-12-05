#!/bin/bash
# CI/CD validation script - runs all checks

set -e

echo "=========================================="
echo "Running CI/CD checks for LinkGen AI"
echo "=========================================="
echo ""

# 1. Validate Docker configurations
echo "Step 1/5: Validating Docker configurations..."
./scripts/validate-docker.sh
echo ""

# 2. Run linters
echo "Step 2/5: Running linters..."
./scripts/lint.sh
echo ""

# 3. Run unit and integration tests
echo "Step 3/5: Running tests..."
cd src
go test -v -race -coverprofile=../coverage.out ./...
cd ..
echo ""

# 4. Check test coverage
echo "Step 4/5: Checking test coverage..."
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
COVERAGE_INT=${COVERAGE%.*}

echo "Total coverage: ${COVERAGE}%"
if [ "$COVERAGE_INT" -lt 80 ]; then
    echo "❌ Coverage is below 80% threshold"
    exit 1
else
    echo "✅ Coverage meets 80% threshold"
fi
echo ""

# 5. Format check
echo "Step 5/5: Checking code formatting..."
cd src
UNFORMATTED=$(gofmt -l .)
if [ -n "$UNFORMATTED" ]; then
    echo "❌ The following files need formatting:"
    echo "$UNFORMATTED"
    exit 1
else
    echo "✅ All files are properly formatted"
fi
cd ..

echo ""
echo "=========================================="
echo "🎉 All CI/CD checks passed!"
echo "=========================================="
