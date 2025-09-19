#!/bin/bash

# Build and deployment script for Chat Room System
set -e

PROJECT_NAME="chatroom"
VERSION=${1:-latest}
PORT=${2:-9090}

echo "🏗️  Building Chat Room System..."
echo "📦 Project: $PROJECT_NAME"
echo "🏷️  Version: $VERSION" 
echo "🚪 Port: $PORT"
echo ""

# Function to check if Docker is running
check_docker() {
    if ! docker info > /dev/null 2>&1; then
        echo "❌ Docker is not running. Please start Docker and try again."
        exit 1
    fi
}

# Function to build Docker image
build_image() {
    echo "🔨 Building Docker image..."
    docker build -t "${PROJECT_NAME}:${VERSION}" .
    echo "✅ Docker image built successfully!"
}

# Function to run container
run_container() {
    echo "🚀 Starting container..."
    
    # Stop existing container if running
    if docker ps -q -f name="${PROJECT_NAME}-server" | grep -q .; then
        echo "🛑 Stopping existing container..."
        docker stop "${PROJECT_NAME}-server"
        docker rm "${PROJECT_NAME}-server"
    fi
    
    # Run new container
    docker run -d \
        --name "${PROJECT_NAME}-server" \
        -p "${PORT}:9090" \
        -e PORT=9090 \
        --restart unless-stopped \
        "${PROJECT_NAME}:${VERSION}"
    
    echo "✅ Container started successfully!"
    echo "🌐 Server available at: http://localhost:${PORT}"
    echo "📡 WebSocket endpoint: ws://localhost:${PORT}/ws"
    echo "🏥 Health check: http://localhost:${PORT}/health"
}

# Function to show logs
show_logs() {
    echo "📋 Container logs:"
    docker logs -f "${PROJECT_NAME}-server"
}

# Main execution
case "${3:-run}" in
    "build")
        check_docker
        build_image
        ;;
    "run")
        check_docker
        build_image
        run_container
        ;;
    "logs")
        show_logs
        ;;
    "stop")
        echo "🛑 Stopping container..."
        docker stop "${PROJECT_NAME}-server" || true
        docker rm "${PROJECT_NAME}-server" || true
        echo "✅ Container stopped!"
        ;;
    "restart")
        check_docker
        echo "🔄 Restarting container..."
        docker stop "${PROJECT_NAME}-server" || true
        docker rm "${PROJECT_NAME}-server" || true
        build_image
        run_container
        ;;
    *)
        echo "Usage: $0 [version] [port] [command]"
        echo ""
        echo "Commands:"
        echo "  run     - Build and run container (default)"
        echo "  build   - Build Docker image only"
        echo "  logs    - Show container logs"
        echo "  stop    - Stop and remove container"  
        echo "  restart - Restart container with rebuild"
        echo ""
        echo "Examples:"
        echo "  $0                    # Build and run on port 9090"
        echo "  $0 v1.0 9090 run      # Build and run version v1.0 on port 9090"
        echo "  $0 latest 9090 build  # Build only"
        echo "  $0 latest 9090 logs   # Show logs"
        exit 1
        ;;
esac
