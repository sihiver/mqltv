#!/bin/bash

echo "=== Testing Create Manual Playlist API ==="
echo ""

# Login as admin
echo "1. Login as admin..."
RESPONSE=$(curl -s -c /tmp/cookies.txt -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"459922"}')

COOKIE=$(grep "admin-session" /tmp/cookies.txt | awk '{print $7}')

if [ -z "$COOKIE" ]; then
  echo "❌ Failed to login"
  exit 1
fi
echo "✅ Login successful"
echo ""

# Create manual playlist
echo "2. Creating manual playlist..."
curl -s -X POST "http://localhost:8080/api/playlists" \
  -H "Cookie: admin-session=$COOKIE" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Manual Playlist",
    "type": "manual"
  }' | jq '.'
echo ""

# Create M3U type playlist
echo "3. Creating M3U type playlist..."
curl -s -X POST "http://localhost:8080/api/playlists" \
  -H "Cookie: admin-session=$COOKIE" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test M3U Playlist",
    "type": "m3u"
  }' | jq '.'
echo ""

# List all playlists
echo "4. Listing all playlists..."
curl -s -X GET "http://localhost:8080/api/playlists" \
  -H "Cookie: admin-session=$COOKIE" | jq '.'
echo ""

echo "=== Test Complete ==="
