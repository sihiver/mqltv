# FFmpeg Integration Guide

## Ringkasan
IPTV Panel sekarang menggunakan **FFmpeg** untuk semua streaming, memastikan bahwa **semua traffic dari client ke provider melalui panel server**, bukan direct ke provider.

## Arsitektur

### Sebelum (Direct HTTP Proxy):
```
Client → Panel Server → HTTP Redirect/Proxy → Provider
```
Masalah: Client kadang langsung redirect ke provider source URL

### Sesudah (FFmpeg-based):
```
Client → Panel Server (FFmpeg) → Provider
```
Semua client streaming **hanya** melalui panel server, FFmpeg mengambil dari provider.

## Cara Kerja FFmpeg Manager

### 1. Multi-Client Single Stream
- Satu channel ditonton banyak client = **satu koneksi FFmpeg ke provider**
- FFmpeg output di-broadcast ke semua client melalui channels
- Menghemat bandwidth hingga **90%+**

### 2. On-Demand Auto Start/Stop
- FFmpeg **otomatis start** saat client pertama connect
- FFmpeg **otomatis stop** setelah 60 detik tidak ada client (idle timeout)
- Resource efficient - hanya berjalan saat diperlukan

### 3. Format Support
- **mpegts** (default): MPEG-TS streaming, cocok untuk live TV
- **hls**: HLS transcoding dengan segmentasi
- **copy**: Copy codec tanpa transcode (paling efficient)

### 4. Failover Support
- Jika source URL pertama gagal, otomatis coba URL berikutnya
- Seamless fallback untuk reliability

## FFmpeg Command yang Digunakan

### MPEG-TS (Default):
```bash
ffmpeg -re -i <SOURCE_URL> \
       -c copy \
       -f mpegts \
       -tune zerolatency \
       pipe:1
```

### HLS:
```bash
ffmpeg -re -i <SOURCE_URL> \
       -c copy \
       -f hls \
       -hls_time 4 \
       -hls_list_size 5 \
       -hls_flags delete_segments \
       pipe:1
```

## Testing

### 1. Test dengan VLC (MPEG-TS Stream)
```bash
# Ambil stream relay
vlc http://localhost:8080/stream/relay1

# Atau channel langsung
vlc http://localhost:8080/api/proxy/channel/1
```

### 2. Test dengan curl
```bash
# Lihat apakah data streaming
curl -v http://localhost:8080/stream/relay1 | head -c 10000 > test.ts

# Verify file
file test.ts  # Should show: MPEG transport stream data
```

### 3. Test Multi-Client
```bash
# Terminal 1
vlc http://localhost:8080/stream/relay1

# Terminal 2
vlc http://localhost:8080/stream/relay1

# Check logs - hanya 1 FFmpeg process harus running
tail -f /tmp/iptv.log
```

### 4. Verify Traffic Tidak Direct ke Provider
```bash
# Monitor koneksi saat streaming
netstat -tupn | grep :8080

# Seharusnya:
# - Client connect ke localhost:8080
# - Panel connect ke provider
# - TIDAK ADA direct connection dari client ke provider
```

### 5. Test Auto Start/Stop
```bash
# Start stream
vlc http://localhost:8080/stream/relay1 &

# Check FFmpeg running
ps aux | grep ffmpeg

# Stop VLC, tunggu 60 detik
pkill vlc
sleep 65

# Check FFmpeg sudah stop
ps aux | grep ffmpeg  # Should be empty
```

## Monitoring Streams

### API Endpoint
```bash
# Get all active FFmpeg sessions
curl http://localhost:8080/api/streams/status

# Response example:
{
  "total_streams": 2,
  "streams": [
    {
      "id": "relay1",
      "clients": 3,
      "uptime": "5m30s",
      "started_at": "2025-12-08T19:50:00Z"
    }
  ]
}
```

### Check Logs
```bash
# Real-time logs
tail -f /tmp/iptv.log

# Filter FFmpeg only
tail -f /tmp/iptv.log | grep FFmpeg

# Filter start/stop events
tail -f /tmp/iptv.log | grep "Starting FFmpeg\|Stopping FFmpeg"
```

## Troubleshooting

### FFmpeg Not Found
```bash
# Check FFmpeg installed
which ffmpeg

# Install if needed (Ubuntu/Debian)
sudo apt-get install ffmpeg

# Install (macOS)
brew install ffmpeg
```

### Stream Lag or Buffer
Ganti format dari "mpegts" ke "copy" di kode untuk zero-latency:
```go
session := ffmpegManager.GetOrCreateFFmpegSession(sessionID, urls, "copy")
```

### High CPU Usage
FFmpeg transcoding menggunakan CPU. Untuk mengurangi:
1. Gunakan format "copy" (no transcode)
2. Kurangi jumlah concurrent streams
3. Upgrade CPU atau gunakan hardware acceleration

### Connection Refused
```bash
# Check server running
curl http://localhost:8080/api/stats

# Check firewall
sudo ufw status

# Check port listening
netstat -tupn | grep :8080
```

## Performance Tips

### 1. Hardware Acceleration
Untuk server dengan GPU, enable hardware encoding:
```go
// Modify ffmpeg.go startFFmpeg()
args := []string{
    "-hwaccel", "cuda",  // or "vaapi", "qsv"
    "-i", sourceURL,
    // ... rest of args
}
```

### 2. Bandwidth Optimization
- Multi-client: 90%+ bandwidth saving
- On-demand: Resources only used when needed
- Use "copy" format to avoid transcoding overhead

### 3. Scaling
- Each FFmpeg process handles unlimited clients (broadcast model)
- Typical server: 50-100 concurrent streams dengan CPU modern
- Memory: ~50MB per active FFmpeg session

## Keuntungan FFmpeg-based Streaming

✅ **Full Control**: Semua traffic melalui panel server  
✅ **Better Compatibility**: FFmpeg mendukung hampir semua format  
✅ **Transcoding**: Bisa convert format on-the-fly  
✅ **Recording**: Mudah tambah fitur recording  
✅ **Multi-Client Efficient**: Satu source untuk banyak client  
✅ **Resource Smart**: Auto start/stop on-demand  
✅ **Reliable**: Failover ke backup source otomatis  

## Next Steps

Untuk fitur tambahan:
1. **Recording**: Tambah output file ke FFmpeg command
2. **Transcoding Profiles**: Multiple quality options (SD, HD, FHD)
3. **Adaptive Bitrate**: HLS dengan multiple variants
4. **DVR/Timeshift**: Buffer untuk pause/rewind
5. **Analytics**: Track bandwidth, viewer stats per stream
