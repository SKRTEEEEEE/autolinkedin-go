#!/bin/bash
# Build production Docker image

echo "Building LinkGen AI production image..."

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "Error: Docker is not running. Please start Docker and try again."
    exit 1
fi

# Build the production image
docker build --target production -t linkgenai:latest .

if [ $? -eq 0 ]; then
    echo "✅ Production image built successfully!"
    echo ""
    echo "Image: linkgenai:latest"
    echo "To run: docker run -p 8080:8080 --env-file .env linkgenai:latest"
else
    echo "❌ Build failed!"
    exit 1
fi
