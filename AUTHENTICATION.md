# Autentikasi Stream IPTV

## Overview
Sistem autentikasi telah ditambahkan ke semua endpoint streaming untuk memastikan hanya user yang terdaftar dan aktif yang dapat menonton channel.

## Endpoints yang Dilindungi

### 1. `/api/proxy/channel/{id}` (MPEG-TS Stream)
Endpoint untuk streaming channel via FFmpeg dalam format MPEG-TS.

**Parameter Wajib:**
- `username` - Username user yang terdaftar
- `password` - Password user dalam plain text

**Contoh:**
```
http://localhost:8080/api/proxy/channel/123?username=client01&password=pass123
```

### 2. `/api/proxy/channel/{id}/hls` (HLS Stream)
Endpoint untuk streaming channel via FFmpeg dalam format HLS.

**Parameter Wajib:**
- `username` - Username user yang terdaftar
- `password` - Password user dalam plain text

**Contoh:**
```
http://localhost:8080/api/proxy/channel/123/hls?username=client01&password=pass123
```

### 3. `/stream/{path}` (Relay Stream MPEG-TS)
Endpoint untuk streaming relay via FFmpeg dalam format MPEG-TS.

**Parameter Wajib:**
- `username` - Username user yang terdaftar
- `password` - Password user dalam plain text

**Contoh:**
```
http://localhost:8080/stream/relay1?username=client01&password=pass123
```

### 4. `/stream/{path}/hls` (Relay Stream HLS)
Endpoint untuk streaming relay via FFmpeg dalam format HLS.

**Parameter Wajib:**
- `username` - Username user yang terdaftar
- `password` - Password user dalam plain text

**Contoh:**
```
http://localhost:8080/stream/relay1/hls?username=client01&password=pass123
```

## Validasi Autentikasi

Setiap request ke endpoint streaming akan divalidasi dengan:

1. **Credentials Check** - Username dan password harus cocok dengan database
2. **Active Status** - User harus dalam status aktif (`is_active = true`)
3. **Expiration Check** - Jika user memiliki expiration date, harus belum expired

## Error Responses

### 401 Unauthorized
```
Authentication required: username and password parameters missing
```
Terjadi ketika parameter `username` atau `password` tidak disertakan.

```
Invalid username or password
```
Terjadi ketika credentials tidak cocok dengan database.

### 403 Forbidden
```
User account is inactive
```
Terjadi ketika user account tidak aktif (`is_active = false`).

```
User subscription has expired
```
Terjadi ketika user subscription sudah melewati `expires_at` date.

## Generate Playlist dengan Autentikasi

Saat generate playlist untuk user:

1. Pilih user dari dropdown
2. Pilih channels yang ingin disertakan
3. Sistem akan meminta input password user
4. Password akan disertakan dalam URL stream di M3U file

**Format M3U Generated:**
```
#EXTM3U
# IPTV Playlist for: client01
# Generated: 09/12/2025 10:30:00
# Total Channels: 50

#EXTINF:-1 tvg-id="1" tvg-name="Channel 1" group-title="News",Channel 1
http://localhost:8080/api/proxy/channel/1?username=client01&password=pass123
```

## Security Notes

⚠️ **Perhatian Keamanan:**

1. Password dikirim dalam **plain text** melalui URL query parameter
2. **Gunakan HTTPS** untuk production environment untuk encrypt traffic
3. Password di-hash dengan **MD5** di database
4. Pastikan playlist M3U tidak dibagikan ke orang lain karena berisi credentials

## Keuntungan Sistem Autentikasi

✅ **User yang dihapus otomatis tidak bisa akses** - Kredensial langsung invalid
✅ **User yang expired otomatis terblokir** - Sistem cek expiration date setiap request
✅ **User yang di-suspend bisa langsung terblokir** - Set `is_active = false`
✅ **Tracking per-user** - Bisa track siapa yang nonton apa

## Testing

### Test dengan User Valid:
```bash
curl "http://localhost:8080/api/proxy/channel/1?username=client01&password=pass123"
```

### Test tanpa Autentikasi (akan error):
```bash
curl "http://localhost:8080/api/proxy/channel/1"
```

### Test dengan User yang Dihapus (akan error):
```bash
curl "http://localhost:8080/api/proxy/channel/1?username=deleted_user&password=pass123"
```

## Migration dari Playlist Lama

Jika Anda memiliki playlist yang di-generate sebelum implementasi autentikasi:

1. **Re-generate semua playlist** dari tab "Generate Playlist"
2. Input password user saat di-prompt
3. Distribute playlist baru ke user
4. Playlist lama tanpa autentikasi akan **gagal stream**

## Admin Panel

Di admin panel (`Users` tab), Anda bisa:

- Klik tombol **"📺 Channels"** untuk melihat channel apa saja yang bisa ditonton user
- File M3U akan di-parse dan ditampilkan grouped by category
- Pastikan playlist sudah di-generate dengan autentikasi sebelum view

## Troubleshooting

### "Authentication required" Error
- Pastikan URL include `username` dan `password` parameters
- Format: `?username=xxx&password=yyy`

### "Invalid username or password" Error
- Pastikan username dan password benar
- Password harus dalam plain text (sistem akan hash dengan MD5)

### "User account is inactive" Error
- User di-suspend oleh admin
- Hubungi admin untuk aktivasi kembali

### "User subscription has expired" Error
- Langganan user sudah habis
- Perpanjang subscription dari admin panel
