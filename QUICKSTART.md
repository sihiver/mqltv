# 📺 IPTV Panel - Quick Start Guide

## 🔧 Instalasi Go

### Linux/WSL (Ubuntu/Debian):
```bash
# Opsi 1: Menggunakan snap
sudo snap install go --classic

# Opsi 2: Menggunakan apt
sudo apt update
sudo apt install golang-go

# Verifikasi instalasi
go version
```

### Windows:
1. Download installer dari: https://golang.org/dl/
2. Jalankan installer dan ikuti petunjuk
3. Buka Command Prompt dan ketik `go version` untuk verifikasi

### macOS:
```bash
# Menggunakan Homebrew
brew install go

# Verifikasi instalasi
go version
```

## 🚀 Menjalankan Aplikasi

### Cara 1: Menggunakan Script (Recommended)

**Linux/WSL:**
```bash
cd /home/dindin/mqltv
./start.sh
```

**Windows:**
```cmd
cd C:\path\to\mqltv
start.bat
```

### Cara 2: Manual

```bash
cd /home/dindin/mqltv

# Download dependencies
go mod download

# Jalankan aplikasi
go run main.go

# Atau build dulu lalu jalankan
go build -o iptv-panel main.go
./iptv-panel
```

## 🌐 Akses Panel

Setelah server berjalan, buka browser dan akses:
```
http://localhost:8080
```

## 📝 Penggunaan Cepat

### 1. Import Playlist M3U

**Lewat Web Interface:**
1. Buka tab "Import M3U"
2. Masukkan nama: "My TV"
3. Masukkan URL M3U: `http://example.com/playlist.m3u`
4. Klik "Import Playlist"

**Lewat API (curl):**
```bash
curl -X POST http://localhost:8080/api/playlists/import \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My TV",
    "url": "http://example.com/playlist.m3u"
  }'
```

### 2. Buat Relay dengan Failover

**Contoh Relay dengan 3 sources:**
```bash
curl -X POST http://localhost:8080/api/relays \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Sport Channel",
    "output_path": "sport",
    "source_urls": [
      "http://primary-server.com/stream.m3u8",
      "http://backup-server.com/stream.m3u8",
      "http://fallback-server.com/stream.m3u8"
    ]
  }'
```

Akses relay di: `http://localhost:8080/stream/sport`

### 3. Streaming Channel

**Lewat Proxy:**
```
http://localhost:8080/api/proxy/channel/{channel_id}
```

**Contoh dengan VLC:**
```bash
vlc http://localhost:8080/api/proxy/channel/1
```

## 🎯 Contoh M3U Playlist Format

```m3u
#EXTM3U
#EXTINF:-1 tvg-logo="http://example.com/logo.png" group-title="Sports",Sport TV 1
http://stream.example.com/sport1.m3u8
#EXTINF:-1 tvg-logo="http://example.com/logo2.png" group-title="Movies",Movie Channel
http://stream.example.com/movies.m3u8
#EXTINF:-1 tvg-logo="http://example.com/logo3.png" group-title="News",News TV
http://stream.example.com/news.m3u8
```

## 🔧 Konfigurasi Port Custom

```bash
# Set custom port
export PORT=3000
go run main.go

# Atau
PORT=3000 go run main.go
```

## 📊 API Endpoints Lengkap

| Method | Endpoint | Deskripsi |
|--------|----------|-----------|
| GET | `/api/stats` | Dashboard statistics |
| GET | `/api/playlists` | List all playlists |
| POST | `/api/playlists/import` | Import M3U playlist |
| DELETE | `/api/playlists/{id}` | Delete playlist |
| GET | `/api/playlists/{id}/channels` | List channels |
| GET | `/api/playlists/{id}/export` | Export to M3U |
| GET | `/api/channels/search?q={query}` | Search channels |
| POST | `/api/channels/{id}/toggle` | Enable/disable channel |
| GET | `/api/proxy/channel/{id}` | Proxy channel stream |
| GET | `/api/relays` | List all relays |
| POST | `/api/relays` | Create relay |
| DELETE | `/api/relays/{id}` | Delete relay |
| GET | `/stream/{path}` | Access relay stream |

## 🐛 Troubleshooting

### Error: "go: command not found"
Install Go terlebih dahulu (lihat bagian Instalasi Go di atas)

### Error: "port already in use"
Ubah port dengan environment variable:
```bash
PORT=3000 go run main.go
```

### Error: "database locked"
Pastikan hanya ada satu instance aplikasi yang berjalan

### Relay tidak bekerja
- Pastikan source URLs valid dan accessible
- Cek logs untuk error messages
- Test manual dengan curl atau VLC

## 💡 Tips & Tricks

1. **Multiple Relay Sources**: Tambahkan 3-5 source URLs untuk reliability maksimal
2. **Channel Groups**: Gunakan group-title di M3U untuk organizing channels
3. **Logo URLs**: Tambahkan tvg-logo untuk tampilan yang lebih baik
4. **Export M3U**: Export untuk digunakan di aplikasi IPTV player lain
5. **Search**: Gunakan search untuk menemukan channel cepat di database besar

## 🔐 Production Deployment

### Menggunakan Systemd (Linux)

1. Buat service file:
```bash
sudo nano /etc/systemd/system/iptv-panel.service
```

2. Isi dengan:
```ini
[Unit]
Description=IPTV Panel Service
After=network.target

[Service]
Type=simple
User=dindin
WorkingDirectory=/home/dindin/mqltv
ExecStart=/home/dindin/mqltv/iptv-panel
Restart=always
Environment="PORT=8080"
Environment="DB_PATH=/home/dindin/mqltv/iptv.db"
# OPTIONAL (biasanya tidak perlu):
# - Jika tidak diset, sistem akan otomatis pakai Host dari request (r.Host).
#   Catatan: kalau kamu akses panel/API dari `localhost`, maka URL playlist yang di-generate
#   memang akan jadi `http://localhost:PORT/...`.
# - Jika pakai reverse proxy (Nginx/Caddy/Cloudflare), pastikan header
#   X-Forwarded-Host dan X-Forwarded-Proto diteruskan.
# - Jika ingin dipaksa selalu absolut ke domain tertentu, pakai PUBLIC_BASE_URL.
#
# Environment="PUBLIC_BASE_URL=https://iptv.yourdomain.com"
# Environment="HOST=YOUR_VPS_IP:8080"
# Ganti YOUR_VPS_IP dengan IP publik VPS (contoh: 203.0.113.10:8080)
# Atau gunakan domain: Environment="HOST=iptv.yourdomain.com"

[Install]
WantedBy=multi-user.target
```

3. Start service:
```bash
sudo systemctl daemon-reload
sudo systemctl enable iptv-panel
sudo systemctl start iptv-panel
sudo systemctl status iptv-panel
```

### Menggunakan Docker (Optional)

Buat Dockerfile di masa depan jika diperlukan.

## 📞 Support

Jika ada masalah atau pertanyaan:
1. Cek README.md untuk dokumentasi lengkap
2. Cek bagian Troubleshooting di atas
3. Review logs aplikasi untuk error messages

---

**Selamat Menggunakan IPTV Panel! 🎉**
