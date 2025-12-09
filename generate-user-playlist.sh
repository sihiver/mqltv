#!/bin/bash

# Generate M3U Playlist for specific user
# Usage: ./generate-user-playlist.sh <username> <password>

USERNAME="${1:-client01}"
PASSWORD="${2:-rahasia123}"

echo "#EXTM3U" > "playlist-${USERNAME}.m3u"
echo "# IPTV Playlist for: ${USERNAME}"
echo "# Generated: $(date)"
echo ""

# Get all active channels
curl -s 'http://localhost:8080/api/channels/search?q=' | jq -r '.[] | select(.active == 1) | 
  "#EXTINF:-1 tvg-id=\"\(.id)\" tvg-name=\"\(.name)\" group-title=\"\(.category)\",\(.name)
http://localhost:8080/api/proxy/channel/\(.id)?user=\(env.USERNAME)&pass=\(env.PASSWORD)"' \
  --arg USERNAME "$USERNAME" --arg PASSWORD "$PASSWORD" >> "playlist-${USERNAME}.m3u"

echo "✅ Playlist generated: playlist-${USERNAME}.m3u"
echo ""
echo "User dapat menggunakan URL ini di VLC/IPTV player:"
echo "http://localhost:8080/api/proxy/channel/<channel_id>?user=${USERNAME}&pass=${PASSWORD}"
