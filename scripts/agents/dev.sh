#!/bin/bash
# Start development environment with Docker

echo "Starting LinkGen AI development environment..."

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "Error: Docker is not running. Please start Docker and try again."
    exit 1
fi

# Start services
docker-compose up -d

# Wait for services to be healthy
echo "Waiting for services to be ready..."
sleep 5

# Show logs
echo "Development environment is running!"
echo "API: http://localhost:8080"
echo "MongoDB: mongodb://localhost:27017"
echo "NATS: nats://localhost:4222"
echo ""
echo "To view logs, run: docker-compose logs -f app"
echo "To stop, run: docker-compose down"
