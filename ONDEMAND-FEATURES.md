# 🚀 On-Demand Streaming & HLS Features

## ✨ Fitur Baru yang Ditambahkan

### 1. 🎯 On-Demand Auto Start/Stop
Stream **otomatis dimulai** hanya ketika ada client yang menonton, dan **otomatis berhenti** ketika tidak ada client (idle timeout 30 detik).

**Keuntungan:**
- ✅ Hemat bandwidth server (stream tidak jalan terus-menerus)
- ✅ Hemat resource CPU dan memory
- ✅ Scalable untuk ratusan channel
- ✅ Stream hanya aktif saat dibutuhkan

### 2. 👥 Multi-Client Single Stream
Banyak client bisa menonton channel yang sama, tetapi **koneksi ke provider hanya 1**.

**Sebelumnya:**
```
Client 1 → Provider (1 stream)
Client 2 → Provider (1 stream)  ❌ Boros!
Client 3 → Provider (1 stream)
= 3 koneksi ke provider
```

**Sekarang:**
```
Client 1 ┐
Client 2 ├→ Server → Provider (1 stream)  ✅ Efisien!
Client 3 ┘
= 1 koneksi ke provider untuk semua client
```

### 3. 📺 HLS Support
Output stream dalam format **HLS (HTTP Live Streaming)** yang kompatibel dengan semua device dan browser.

**Keuntungan:**
- ✅ Kompatibel dengan iOS, Android, Smart TV
- ✅ Adaptive bitrate ready
- ✅ Buffering lebih baik
- ✅ Standard industri

## 🔗 API Endpoints Baru

### Stream Status & Monitoring

#### GET /api/streams/status
Melihat status semua stream yang sedang aktif.

**Response:**
```json
{
  "total_streams": 2,
  "streams": [
    {
      "id": "sport-tv",
      "active": true,
      "clients": 5,
      "source_url": "http://cdn1.com/sport.m3u8",
      "uptime_seconds": 125.5,
      "bytes_streamed": 52428800,
      "last_activity": "2025-12-08T19:20:00Z"
    }
  ]
}
```

#### GET /api/streams/{id}/status
Melihat status stream tertentu.

**Response:**
```json
{
  "id": "sport-tv",
  "active": true,
  "clients": 5,
  "source_url": "http://cdn1.com/sport.m3u8",
  "uptime_seconds": 125.5,
  "bytes_streamed": 52428800,
  "last_activity": "2025-12-08T19:20:00Z"
}
```

### HLS Streaming Endpoints

#### GET /stream/{path}/hls
Mendapatkan HLS playlist (M3U8) untuk relay.

**Contoh:**
```bash
# Akses HLS playlist
http://localhost:8080/stream/sport-tv/hls

# Gunakan di VLC atau player lain
vlc http://localhost:8080/stream/sport-tv/hls
```

#### GET /stream/{path}/hls/{segment}
Mendapatkan HLS segment (TS file).

**Contoh:**
```
http://localhost:8080/stream/sport-tv/hls/segment_0.ts
```

#### GET /api/proxy/channel/{id}/hls
Mendapatkan HLS playlist untuk channel tertentu.

**⚠️ PENTING:** Endpoint ini hanya bekerja jika source channel sudah dalam format HLS (.m3u8)

**Contoh:**
```bash
# Channel ID 1 sebagai HLS (jika source-nya .m3u8)
http://localhost:8080/api/proxy/channel/1/hls

# Gunakan di player
vlc http://localhost:8080/api/proxy/channel/1/hls

# Jika source BUKAN HLS, gunakan direct proxy:
vlc http://localhost:8080/api/proxy/channel/1
```

## 📝 Cara Menggunakan

### 1. Stream Relay (Multi-Client)

**Setup Relay:**
```bash
curl -X POST http://localhost:8080/api/relays \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Sport TV",
    "output_path": "sport-tv",
    "source_urls": [
      "http://cdn1.com/sport.m3u8",
      "http://cdn2.com/sport.m3u8"
    ]
  }'
```

**Cara Nonton:**

**Direct Stream:**
```
http://localhost:8080/stream/sport-tv
```

**HLS Stream:**
```
http://localhost:8080/stream/sport-tv/hls
```

**Multiple Clients:**
```bash
# Client 1
vlc http://localhost:8080/stream/sport-tv/hls

# Client 2 (menggunakan stream yang sama!)
vlc http://localhost:8080/stream/sport-tv/hls

# Client 3
vlc http://localhost:8080/stream/sport-tv/hls

# Server hanya membuka 1 koneksi ke provider!
```

### 2. Monitoring Stream

**Cek semua stream aktif:**
```bash
curl http://localhost:8080/api/streams/status | jq
```

**Output:**
```json
{
  "total_streams": 1,
  "streams": [
    {
      "id": "sport-tv",
      "active": true,
      "clients": 3,
      "uptime_seconds": 45.2
    }
  ]
}
```

### 3. Auto Start/Stop Demo

**Scenario:**
```bash
# 1. Tidak ada yang nonton - stream idle/off
curl http://localhost:8080/api/streams/status
# Result: {"total_streams": 0, "streams": []}

# 2. Client 1 mulai nonton
vlc http://localhost:8080/stream/sport-tv/hls &
# Stream otomatis START!

# 3. Cek status
curl http://localhost:8080/api/streams/status
# Result: {"total_streams": 1, "streams": [...]}

# 4. Client 2 & 3 ikut nonton
vlc http://localhost:8080/stream/sport-tv/hls &
vlc http://localhost:8080/stream/sport-tv/hls &
# Tetap 1 stream ke provider!

# 5. Semua client stop
# Tunggu 30 detik...
# Stream otomatis STOP!
```

## 🎬 Embed di Web/App

### HTML5 Video Player (HLS.js)
```html
<!DOCTYPE html>
<html>
<head>
    <script src="https://cdn.jsdelivr.net/npm/hls.js@latest"></script>
</head>
<body>
    <video id="video" controls width="640" height="360"></video>
    <script>
        var video = document.getElementById('video');
        var videoSrc = 'http://localhost:8080/stream/sport-tv/hls';
        
        if (Hls.isSupported()) {
            var hls = new Hls();
            hls.loadSource(videoSrc);
            hls.attachMedia(video);
        } else if (video.canPlayType('application/vnd.apple.mpegurl')) {
            video.src = videoSrc;
        }
    </script>
</body>
</html>
```

### M3U Playlist untuk IPTV Player
```m3u
#EXTM3U
#EXTINF:-1 tvg-logo="logo.png" group-title="Sports",Sport TV HLS
http://localhost:8080/stream/sport-tv/hls

#EXTINF:-1 tvg-logo="logo.png" group-title="Movies",Movie Channel HLS
http://localhost:8080/stream/movies/hls
```

## ⚙️ Konfigurasi

### Idle Timeout (Edit streaming/manager.go)
```go
// Default: 30 detik
idleTimeout: 30 * time.Second,

// Ubah ke 60 detik
idleTimeout: 60 * time.Second,
```

### HLS Segment Duration (Edit streaming/hls.go)
```go
// Default: 4 detik per segment
segmentDur: 4 * time.Second,

// Ubah ke 6 detik
segmentDur: 6 * time.Second,
```

### Buffer Size (Edit streaming/manager.go)
```go
// Default: 2MB ring buffer
buffer: NewRingBuffer(2 * 1024 * 1024),

// Ubah ke 5MB
buffer: NewRingBuffer(5 * 1024 * 1024),
```

## 📊 Performance

### Before (Tanpa On-Demand)
- 100 channels × 24 jam streaming = 2.4TB/hari bandwidth
- 100 channels aktif terus = High CPU usage
- Memory: ~500MB untuk 100 streams

### After (Dengan On-Demand)
- Hanya channel yang ditonton yang streaming
- 10 channels ditonton × 4 jam = 40GB/hari bandwidth ✅ 98% lebih hemat!
- CPU usage hanya untuk stream aktif
- Memory: ~50MB untuk 10 active streams

### Multi-Client Benefits
**10 clients nonton 1 channel:**
- Before: 10 koneksi ke provider = 10× bandwidth cost
- After: 1 koneksi ke provider = 1× bandwidth cost ✅ 90% lebih hemat!

## 🔧 Troubleshooting

### HLS tidak berfungsi
```bash
# Cek apakah folder hls_cache ada
ls -la hls_cache/

# Cek permission
chmod 755 hls_cache/

# Cek logs
tail -f logs/iptv.log
```

### Stream tidak auto-stop
```bash
# Cek apakah masih ada client
curl http://localhost:8080/api/streams/status

# Lihat log untuk idle timeout
grep "idle" logs/iptv.log
```

### Segment not found
```bash
# HLS segments dibuat on-the-fly
# Tunggu beberapa detik setelah start stream
# Segments akan tersedia dalam 5-10 detik
```

## 🎯 Use Cases

### 1. IPTV Provider dengan Banyak Channel
- Import 500 channels dari M3U
- Hanya channel yang ditonton yang streaming
- Hemat bandwidth dan biaya hosting

### 2. Re-streaming dengan Failover
- Setup relay dengan 3 sources
- Multi-client support
- Automatic failover jika source gagal

### 3. Personal IPTV Server
- Stream dari berbagai provider
- HLS untuk compatibility
- Monitor usage dengan API

### 4. Corporate TV Streaming
- Internal TV channels
- Multi-user support
- Resource efficient

## 📚 Technical Details

### Stream Lifecycle
1. **Idle** - Tidak ada client, stream tidak aktif
2. **Starting** - Client pertama connect, mulai connect ke source
3. **Active** - Stream aktif, data mengalir ke clients
4. **Stopping** - Semua client disconnect, tunggu idle timeout
5. **Stopped** - Idle timeout tercapai, stream dihentikan

### Multi-Client Architecture
```
Source Stream (Provider)
        ↓
   Stream Manager
   (Ring Buffer)
        ↓
   ├─→ Client 1
   ├─→ Client 2
   ├─→ Client 3
   └─→ Client N
```

### HLS Segmentation
```
Source Stream
     ↓
HLS Segmenter
     ↓
├─ segment_0.ts (4s)
├─ segment_1.ts (4s)
├─ segment_2.ts (4s)
└─ playlist.m3u8
```

---

**🎉 Sekarang IPTV Panel Anda lebih efisien dan scalable!**
