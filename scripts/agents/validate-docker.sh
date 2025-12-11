#!/bin/bash
# Validate Docker configurations

echo "Validating Docker configurations..."

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed"
    exit 1
fi

# Check if Docker Compose is installed
if ! command -v docker-compose &> /dev/null; then
    echo "❌ Docker Compose is not installed"
    exit 1
fi

echo "✅ Docker and Docker Compose are installed"

# Validate docker-compose.yml syntax
echo "Checking docker-compose.yml syntax..."
if docker-compose -f docker-compose.yml config > /dev/null 2>&1; then
    echo "✅ docker-compose.yml is valid"
else
    echo "❌ docker-compose.yml has syntax errors"
    docker-compose -f docker-compose.yml config
    exit 1
fi

# Validate docker-compose.test.yml syntax
echo "Checking docker-compose.test.yml syntax..."
if docker-compose -f docker-compose.test.yml config > /dev/null 2>&1; then
    echo "✅ docker-compose.test.yml is valid"
else
    echo "❌ docker-compose.test.yml has syntax errors"
    docker-compose -f docker-compose.test.yml config
    exit 1
fi

# Validate Dockerfile syntax
echo "Checking Dockerfile syntax..."
if docker build --target development -t linkgenai-validate-test -f Dockerfile . > /dev/null 2>&1; then
    echo "✅ Dockerfile development stage is valid"
    docker rmi linkgenai-validate-test > /dev/null 2>&1
else
    echo "❌ Dockerfile has syntax errors in development stage"
    exit 1
fi

echo ""
echo "🎉 All Docker configurations are valid!"
exit 0
