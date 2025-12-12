#!/bin/bash

# Install Go jika belum terinstall
if ! command -v go &> /dev/null; then
    echo "Go tidak terinstall. Menginstall Go..."
    echo "Silakan jalankan salah satu command berikut:"
    echo "  sudo snap install go"
    echo "  sudo apt install golang-go"
    exit 1
fi

# Download dependencies
echo "📦 Downloading dependencies..."
go mod download

# Build aplikasi
echo "🔨 Building application..."
go build -o iptv-panel main.go

# Check health check config
if [ "$DISABLE_HEALTH_CHECK" = "1" ]; then
    echo "⚠️  Auto health check: DISABLED (manual only)"
else
    echo "✅ Auto health check: ENABLED (30 min, max 3 concurrent)"
fi

# Jalankan aplikasi
echo "🚀 Starting IPTV Panel..."
./iptv-panel
