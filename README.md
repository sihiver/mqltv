# IPTV Panel

Panel manajemen IPTV dengan fitur import playlist M3U, relay streaming dengan failover, dan proxy multiple sources.

## 🌟 Fitur

### Fitur Utama
- ✅ **Import M3U Playlist** - Import playlist dari URL M3U
- ✅ **Manajemen Channels** - Aktifkan/nonaktifkan channel, cari channel
- ✅ **Stream Relay** - Relay streaming dengan multiple sources untuk failover
- ✅ **Proxy Streaming** - Proxy channel streaming melalui server
- ✅ **Export M3U** - Export playlist ke format M3U
- ✅ **Dashboard** - Statistik real-time playlist dan channel
- ✅ **SQLite Database** - Database lokal yang ringan

### 🚀 Fitur Advanced (NEW!)
- ✅ **On-Demand Auto Start/Stop** - Stream otomatis mulai saat ada viewer, stop saat idle
- ✅ **Multi-Client Single Stream** - Banyak client, hanya 1 koneksi ke provider (hemat 90%+ bandwidth!)
- ✅ **HLS Support** - Output HLS untuk compatibility dengan semua device
- ✅ **Stream Monitoring** - Real-time monitoring jumlah viewer dan bandwidth
- ✅ **Resource Efficient** - CPU & bandwidth hanya digunakan saat ada yang nonton

## 🚀 Cara Install

### Persyaratan
- Go 1.21 atau lebih baru
- SQLite3

### Instalasi

1. Clone atau download project ini:
```bash
cd /home/dindin/mqltv
```

2. Download dependencies:
```bash
go mod download
```

3. Jalankan aplikasi:
```bash
go run main.go
```

4. Buka browser dan akses:
```
http://localhost:8080
```

## 📖 Cara Menggunakan

### 1. Import Playlist M3U
- Klik tab "Import M3U"
- Masukkan nama playlist dan URL M3U
- Klik "Import Playlist"
- Playlist akan otomatis ter-parse dan channels akan tersimpan

### 2. Kelola Channels
- Klik tab "Channels" atau "Playlists" → "Lihat Channels"
- Enable/disable channel sesuai kebutuhan
- Gunakan fitur search untuk mencari channel
- Klik "Play" untuk streaming channel

### 3. Buat Stream Relay
- Klik tab "Relays"
- Klik "Buat Relay Baru"
- Masukkan nama relay dan output path
- Tambahkan multiple source URLs untuk failover
- Relay akan otomatis switch ke source berikutnya jika source pertama gagal

### 4. Export Playlist
- Klik "Export M3U" pada playlist yang diinginkan
- File M3U akan didownload

## 🔗 API Endpoints

### Playlists
- `GET /api/playlists` - Daftar semua playlists
- `POST /api/playlists/import` - Import M3U playlist
- `DELETE /api/playlists/{id}` - Hapus playlist
- `GET /api/playlists/{id}/channels` - Daftar channels dalam playlist
- `GET /api/playlists/{id}/export` - Export playlist ke M3U

### Channels
- `GET /api/channels/search?q={query}` - Cari channels
- `POST /api/channels/{id}/toggle` - Toggle status channel
- `GET /api/proxy/channel/{id}` - Proxy stream channel

### Relays
- `GET /api/relays` - Daftar semua relays
- `POST /api/relays` - Buat relay baru
- `DELETE /api/relays/{id}` - Hapus relay
- `GET /stream/{path}` - Stream relay endpoint

### Stats
- `GET /api/stats` - Dashboard statistics

## 🛠️ Konfigurasi

### Environment Variables
```bash
# Port server (default: 8080)
export PORT=8080

# Database path (default: ./iptv.db)
export DB_PATH=/path/to/iptv.db
```

## 📂 Struktur Project

```
/home/dindin/mqltv/
├── main.go                 # Entry point aplikasi
├── go.mod                  # Go module dependencies
├── database/
│   └── db.go              # Database initialization & schema
├── models/
│   └── models.go          # Data models
├── handlers/
│   └── handlers.go        # HTTP handlers & API logic
├── parser/
│   └── m3u.go            # M3U parser
└── static/
    └── index.html        # Web frontend
```

## 🔧 Build untuk Production

```bash
# Build binary
go build -o iptv-panel main.go

# Jalankan
./iptv-panel
```

## 📝 Contoh Request

### Import Playlist
```bash
curl -X POST http://localhost:8080/api/playlists/import \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My IPTV",
    "url": "http://example.com/playlist.m3u"
  }'
```

### Buat Relay
```bash
curl -X POST http://localhost:8080/api/relays \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Sport TV Relay",
    "output_path": "sport-tv",
    "source_urls": [
      "http://source1.com/stream.m3u8",
      "http://source2.com/stream.m3u8"
    ]
  }'
```

## 🎯 Fitur Relay Failover

Sistem relay mendukung multiple sources dengan automatic failover:
- Jika source pertama gagal, otomatis switch ke source kedua
- Terus mencoba semua sources sampai ada yang berhasil
- Ideal untuk streaming yang reliable dengan backup sources

## 📄 License

MIT License - Silakan digunakan dan dimodifikasi sesuai kebutuhan.

## 🤝 Kontribusi

Kontribusi sangat diterima! Silakan buat pull request atau laporkan issue.

---

Dibuat dengan ❤️ menggunakan Go & HTML
