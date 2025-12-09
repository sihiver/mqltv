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

# Jalankan aplikasi
echo "🚀 Starting IPTV Panel..."
./iptv-panel
