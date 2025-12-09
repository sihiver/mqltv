#!/bin/bash

# IPTV Panel Test Script
# Skrip untuk testing API endpoints

BASE_URL="http://localhost:8080"

echo "🧪 Testing IPTV Panel API..."
echo "================================"
echo ""

# Test 1: Check if server is running
echo "1️⃣  Testing server health..."
if curl -s "$BASE_URL/api/stats" > /dev/null; then
    echo "✅ Server is running"
else
    echo "❌ Server is not running. Please start the server first."
    exit 1
fi
echo ""

# Test 2: Get statistics
echo "2️⃣  Getting dashboard stats..."
curl -s "$BASE_URL/api/stats" | python3 -m json.tool
echo ""

# Test 3: Import playlist
echo "3️⃣  Testing playlist import..."
curl -s -X POST "$BASE_URL/api/playlists/import" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Playlist",
    "url": "file:///home/dindin/mqltv/example-playlist.m3u"
  }' | python3 -m json.tool
echo ""

# Test 4: Get all playlists
echo "4️⃣  Getting all playlists..."
curl -s "$BASE_URL/api/playlists" | python3 -m json.tool
echo ""

# Test 5: Search channels
echo "5️⃣  Searching for channels..."
curl -s "$BASE_URL/api/channels/search?q=sport" | python3 -m json.tool
echo ""

# Test 6: Create relay
echo "6️⃣  Creating test relay..."
curl -s -X POST "$BASE_URL/api/relays" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Relay",
    "output_path": "test-relay",
    "source_urls": [
      "http://example1.com/stream.m3u8",
      "http://example2.com/stream.m3u8"
    ]
  }' | python3 -m json.tool
echo ""

# Test 7: Get all relays
echo "7️⃣  Getting all relays..."
curl -s "$BASE_URL/api/relays" | python3 -m json.tool
echo ""

echo "================================"
echo "✅ Testing completed!"
echo ""
echo "📝 Manual Tests:"
echo "  - Open browser: http://localhost:8080"
echo "  - Test import: Use example-playlist.m3u"
echo "  - Test relay: Create relay and access /stream/{path}"
echo ""
