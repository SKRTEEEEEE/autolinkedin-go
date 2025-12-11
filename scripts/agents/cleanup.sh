#!/bin/bash
# Clean up Docker resources and build artifacts

echo "Cleaning up LinkGen AI Docker resources..."

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "Warning: Docker is not running. Skipping Docker cleanup."
else
    # Stop and remove development containers
    echo "Stopping development containers..."
    docker-compose down

    # Stop and remove test containers with volumes
    echo "Stopping test containers and removing volumes..."
    docker-compose -f docker-compose.test.yml down -v

    # Remove dangling volumes (optional)
    read -p "Remove dangling volumes? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        docker volume prune -f
        echo "Dangling volumes removed."
    fi

    # Remove dangling images (optional)
    read -p "Remove dangling images? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        docker image prune -f
        echo "Dangling images removed."
    fi

    echo "✅ Docker cleanup complete!"
fi

# Clean Go build cache
echo "Cleaning Go build cache..."
cd src && go clean -cache -testcache

# Remove build artifacts
echo "Removing build artifacts..."
cd ..
rm -rf bin/
rm -f coverage.out coverage.html

echo "✅ Cleanup complete!"
