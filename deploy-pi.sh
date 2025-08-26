#!/bin/bash

# KAELO Service Deployment Script for Raspberry Pi
# This script pulls and runs the latest Docker image from GHCR

set -e

IMAGE_NAME="ghcr.io/yourusername/kaelo-service"
CONTAINER_NAME="kaelo-monitoring"

echo "🚀 Starting KAELO Service deployment on Raspberry Pi..."

# Stop and remove existing container if it exists
if docker ps -a --format 'table {{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    echo "🛑 Stopping existing container..."
    docker stop ${CONTAINER_NAME} || true
    docker rm ${CONTAINER_NAME} || true
fi

# Pull the latest image
echo "📥 Pulling latest image from GHCR..."
docker pull ${IMAGE_NAME}:main

# Run the container
echo "🏃 Starting KAELO monitoring service..."
docker run -d \
    --name ${CONTAINER_NAME} \
    --restart unless-stopped \
    --memory="256m" \
    --cpus="0.5" \
    ${IMAGE_NAME}:main

# Check if container is running
if docker ps --format 'table {{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    echo "✅ KAELO Service deployed successfully!"
    echo "📊 Container status:"
    docker ps --filter name=${CONTAINER_NAME} --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
    echo ""
    echo "📝 To view logs: docker logs -f ${CONTAINER_NAME}"
    echo "🛑 To stop: docker stop ${CONTAINER_NAME}"
else
    echo "❌ Deployment failed!"
    echo "📝 Check logs: docker logs ${CONTAINER_NAME}"
    exit 1
fi
