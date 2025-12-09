# Admin Panel Authentication

Sistem login untuk mengamankan akses admin panel IPTV.

## Default Login

```
Username: admin
Password: admin123
```

**⚠️ PENTING: Ganti password setelah instalasi pertama!**

## Cara Menggunakan

### Login
1. Akses: `http://localhost:8080/login.html`
2. Masukkan username & password
3. Klik "🔐 Login"

### Logout
Klik tombol "🚪 Logout" di pojok kanan atas

## API Endpoints

**POST /api/auth/login** - Login
```json
{"username": "admin", "password": "admin123"}
```

**POST /api/auth/logout** - Logout

**GET /api/auth/check** - Cek status login

## Ganti Password

```bash
# 1. Generate MD5 hash password baru
echo -n "password_baru" | md5sum

# 2. Update database
sqlite3 iptv.db "UPDATE admins SET password = 'MD5_HASH' WHERE username = 'admin'"
```

## Security

- Session berlaku 7 hari
- Semua endpoint `/api/*` (kecuali auth) dilindungi
- Cookie HttpOnly untuk keamanan
- Redirect otomatis ke login jika belum login

## Tambah Admin Baru

```bash
# Generate password hash
echo -n "newpassword" | md5sum

# Insert ke database
sqlite3 iptv.db "INSERT INTO admins (username, password, full_name, is_active) VALUES ('newadmin', 'MD5_HASH', 'Full Name', 1)"
```

## Troubleshooting

**Reset password admin:**
```bash
sqlite3 iptv.db "UPDATE admins SET password = '0192023a7bbd73250516f069df18b500' WHERE username = 'admin'"
```
(Password: admin123)
