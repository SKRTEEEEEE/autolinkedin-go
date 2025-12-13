#!/bin/bash

# Reset Development Data Script
# This script clears all development data to force reseeding

echo "🔄 Resetting LinkGen AI development data..."

# Stop and remove MongoDB volume (this will clear all data)
echo "🧹 Clearing MongoDB data..."
docker-compose down -v mongodb-data

# Restart the services
echo "🚀 Restarting services..."
docker-compose up -d

echo "✅ Development data has been reset!"
echo "📝 The application will now reseed with fresh data when it starts."
echo "🌐 Application will be available at http://localhost:8080"
