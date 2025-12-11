#!/bin/bash
# Run tests in isolated Docker environment

echo "Running tests in isolated Docker environment..."

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "Error: Docker is not running. Please start Docker and try again."
    exit 1
fi

# Run tests with automatic cleanup
docker-compose -f docker-compose.test.yml up --abort-on-container-exit --exit-code-from app

# Capture exit code
EXIT_CODE=$?

# Clean up ephemeral containers and volumes
echo "Cleaning up test containers and volumes..."
docker-compose -f docker-compose.test.yml down -v

if [ $EXIT_CODE -eq 0 ]; then
    echo "✅ All tests passed!"
else
    echo "❌ Tests failed with exit code $EXIT_CODE"
fi

exit $EXIT_CODE
