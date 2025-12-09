# User Management Guide

## Overview
Sistem user management memungkinkan Anda membuat akun untuk client dengan:
- Username & password
- Masa aktif (expired date)
- Limit koneksi bersamaan (max connections)
- Tracking koneksi aktif
- Auto-expire subscription

## Cara Penggunaan

### 1. Via Web Interface (Recommended)

1. Buka browser: `http://localhost:8080`
2. Klik tab **Users**
3. Klik tombol **"➕ Tambah User"**
4. Isi form:
   - **Username**: username untuk login (unique)
   - **Password**: password untuk akses stream
   - **Nama Lengkap**: nama lengkap client
   - **Email**: email client (opsional)
   - **Max Koneksi**: berapa device bisa nonton bersamaan (1-10)
   - **Durasi (Hari)**: lama subscription (30 = 30 hari, 0 = unlimited)
   - **Catatan**: notes tambahan
5. Klik **"Buat User"**

#### Fitur Lain di Web:
- **Edit User**: Update info & extend subscription
- **Reset Password**: Ganti password user
- **Hapus User**: Hapus user & semua sessions
- **Lihat Status**: 
  - ✅ Aktif (masih valid)
  - ⚠️ Akan Expired (< 7 hari)
  - ❌ Expired (sudah lewat)
  - ⏸️ Nonaktif (disabled)

### 2. Via API

#### Create User
```bash
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{
    "username": "client01",
    "password": "rahasia123",
    "full_name": "Client Pertama",
    "email": "client01@example.com",
    "max_connections": 2,
    "duration_days": 30,
    "notes": "Paket Premium"
  }'
```

#### List All Users
```bash
curl http://localhost:8080/api/users | jq '.'
```

#### Update User
```bash
curl -X PUT http://localhost:8080/api/users/1 \
  -H "Content-Type: application/json" \
  -d '{
    "full_name": "Client Updated",
    "email": "new@example.com",
    "max_connections": 3,
    "is_active": true,
    "extend_days": 7,
    "notes": "Extended 7 days"
  }'
```

#### Extend Subscription
```bash
# Extend 30 hari dari expiry date sekarang
curl -X PUT http://localhost:8080/api/users/1 \
  -H "Content-Type: application/json" \
  -d '{
    "extend_days": 30,
    "max_connections": 2,
    "is_active": true
  }'
```

#### Reset Password
```bash
curl -X POST http://localhost:8080/api/users/1/reset-password \
  -H "Content-Type: application/json" \
  -d '{
    "new_password": "newpassword123"
  }'
```

#### Get User Active Connections
```bash
curl http://localhost:8080/api/users/1/connections | jq '.'
```

#### Delete User
```bash
curl -X DELETE http://localhost:8080/api/users/1
```

### 3. Generate Playlist untuk User

Jalankan script:
```bash
./generate-user-playlist.sh client01 rahasia123
```

Ini akan generate file `playlist-client01.m3u` yang bisa diberikan ke client.

### 4. URL Format untuk Client

Client bisa akses stream dengan format:
```
http://YOUR_SERVER_IP:8080/api/proxy/channel/<CHANNEL_ID>?user=USERNAME&pass=PASSWORD
```

Contoh:
```
http://192.168.1.100:8080/api/proxy/channel/25?user=client01&pass=rahasia123
```

## Use Cases

### Scenario 1: Paket Basic (1 Device, 30 Hari)
```bash
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{
    "username": "basic01",
    "password": "basic123",
    "full_name": "Basic Package User",
    "max_connections": 1,
    "duration_days": 30,
    "notes": "Basic - 1 device"
  }'
```

### Scenario 2: Paket Premium (5 Devices, 90 Hari)
```bash
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{
    "username": "premium01",
    "password": "premium123",
    "full_name": "Premium Package User",
    "max_connections": 5,
    "duration_days": 90,
    "notes": "Premium - 5 devices"
  }'
```

### Scenario 3: Trial (2 Devices, 7 Hari)
```bash
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{
    "username": "trial01",
    "password": "trial123",
    "full_name": "Trial User",
    "max_connections": 2,
    "duration_days": 7,
    "notes": "7 Days Trial"
  }'
```

### Scenario 4: Unlimited/Lifetime
```bash
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{
    "username": "lifetime01",
    "password": "lifetime123",
    "full_name": "Lifetime User",
    "max_connections": 3,
    "duration_days": 0,
    "notes": "Lifetime subscription"
  }'
```

## Database Schema

### Users Table
```sql
- id: Primary key
- username: Unique username
- password: MD5 hashed password
- full_name: Full name
- email: Email address
- max_connections: Max concurrent connections
- is_active: Active status (1/0)
- created_at: Creation timestamp
- activated_at: Activation timestamp
- expires_at: Expiry timestamp (NULL = unlimited)
- last_login: Last login timestamp
- notes: Additional notes
```

### User Sessions Table (Future)
```sql
- id: Primary key
- user_id: Foreign key to users
- token: Session token
- ip_address: Client IP
- user_agent: Client user agent
- created_at: Session start
- expires_at: Session expiry
```

### User Connections Table
```sql
- id: Primary key
- user_id: Foreign key to users
- channel_id: Current watching channel
- ip_address: Client IP
- connected_at: Connection start
- disconnected_at: Connection end
```

## Monitoring

### Check Users Status
```bash
# Via API
curl http://localhost:8080/api/users | jq '.[] | {username, days_remaining, is_expired, is_active}'

# Via Web
# Go to Users tab, see status badges
```

### Check Active Connections per User
```bash
curl http://localhost:8080/api/users/<USER_ID>/connections | jq '.'
```

### Check All Active Streams
```bash
curl http://localhost:8080/api/streams/status | jq '.'
```

## Auto Test Script

Run: `./test-user.sh`

Ini akan:
1. Create user baru
2. List semua users
3. Update user & extend subscription
4. Check active connections
5. Reset password

## Tips

1. **Backup Database Reguler**
   ```bash
   cp iptv.db iptv.db.backup-$(date +%Y%m%d)
   ```

2. **Monitor Expired Users**
   ```bash
   curl -s http://localhost:8080/api/users | jq '.[] | select(.is_expired == true)'
   ```

3. **Monitor Users yang Akan Expired (< 7 hari)**
   ```bash
   curl -s http://localhost:8080/api/users | jq '.[] | select(.days_remaining < 7 and .days_remaining > 0)'
   ```

4. **Count Active vs Expired Users**
   ```bash
   echo "Active: $(curl -s http://localhost:8080/api/users | jq '[.[] | select(.is_expired == false and .is_active == true)] | length')"
   echo "Expired: $(curl -s http://localhost:8080/api/users | jq '[.[] | select(.is_expired == true)] | length')"
   ```

## Next Steps (TODO)

- [ ] Authentication middleware untuk stream endpoints
- [ ] Enforce max_connections limit
- [ ] Auto-disable expired users
- [ ] Email notification before expiry
- [ ] User login history
- [ ] Generate per-user playlist endpoint
- [ ] User bandwidth usage tracking
