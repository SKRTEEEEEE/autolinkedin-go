#!/bin/bash
# Clean up Docker resources

echo "Cleaning up LinkGen AI Docker resources..."

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

echo "✅ Cleanup complete!"
