# User Login API (Client / Android)

## Endpoint

```
POST /api/user/login
```

API ini digunakan untuk login **penonton / user** (misalnya aplikasi Android) tanpa membutuhkan cookie admin.

## Authentication

Tidak memerlukan autentikasi admin.

## Request

**Content-Type:** `application/json`

```json
{
  "username": "nizam",
  "password": "123"
}
```

## Response

### Success (Login OK)

**HTTP:** `200`

```json
{
  "code": 0,
  "data": {
    "id": 11,
    "username": "nizam",
    "full_name": "",
    "email": "",
    "max_connections": 1,
    "active_connections": 0,
    "is_active": true,
    "is_expired": false,
    "days_remaining": 66,
    "created_at": "2025-12-14T16:18:54Z",
    "activated_at": "2025-12-14T23:18:54Z",
    "expires_at": "2026-03-14T23:18:54Z",
    "last_login": null,
    "notes": "",
    "valid_credentials": true,
    "playlist_url": "/mql/nizam.m3u"
  },
  "message": "Login successful"
}
```

### Error (Invalid Credentials)

**HTTP:** `401`

```json
{
  "code": 1,
  "data": {
    "username": "nizam",
    "valid_credentials": false
  },
  "message": "Invalid credentials"
}
```

### Error (Account Inactive)

**HTTP:** `403`

```json
{
  "code": 1,
  "data": {
    "username": "nizam",
    "is_active": false,
    "valid_credentials": true
  },
  "message": "User account is inactive"
}
```

### Error (Subscription Expired)

**HTTP:** `403`

```json
{
  "code": 1,
  "data": {
    "username": "nizam",
    "is_expired": true,
    "expires_at": "2025-12-01T00:00:00Z",
    "days_remaining": -37,
    "valid_credentials": true
  },
  "message": "User subscription has expired"
}
```

## Client Flow (Android)

1. Panggil `/api/user/login` saat user menekan tombol login.
2. Jika response `code == 0` → izinkan masuk aplikasi.
3. Jika HTTP `403` dan `message == "User subscription has expired"` (atau `data.is_expired == true`) → arahkan ke halaman informasi perpanjangan paket.

## Notes

- Password user di database tersimpan dalam bentuk hash MD5, tetapi endpoint ini menerima password plain-text.
- Untuk production/remote wajib gunakan HTTPS.
