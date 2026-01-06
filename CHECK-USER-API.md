# Check User API Documentation

## Endpoint

```
GET /api/users/check/{username}
```

API ini digunakan untuk memeriksa keberadaan user dan memvalidasi kredensial (opsional).

## Authentication

Memerlukan autentikasi admin (session cookie).

## Parameters

### Path Parameters
- `username` (required): Username yang akan dicek

### Query Parameters
- `password` (optional): Password user untuk validasi kredensial

## Response

### Success Response (User Found)

**Code**: `0`

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
    "activated_at": "2025-12-14T23:18:54.039518208+07:00",
    "expires_at": "2026-03-14T23:18:54.039518208+07:00",
    "last_login": null,
    "notes": ""
  },
  "message": "User found"
}
```

### Success Response (With Password Validation)

Jika parameter `password` diberikan, response akan menyertakan field tambahan:

```json
{
  "code": 0,
  "data": {
    "id": 11,
    "username": "nizam",
    ...
    "valid_credentials": true
  },
  "message": "User found"
}
```

### Error Response (Invalid Credentials)

**Code**: `1`

```json
{
  "code": 1,
  "data": {
    "id": 11,
    "username": "nizam",
    ...
    "valid_credentials": false
  },
  "message": "Invalid credentials"
}
```

### Error Response (User Not Found)

**Code**: `1`

```json
{
  "code": 1,
  "data": null,
  "message": "User not found"
}
```

## Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | integer | ID user |
| `username` | string | Username |
| `full_name` | string | Nama lengkap user |
| `email` | string | Email user |
| `max_connections` | integer | Maksimal koneksi yang diizinkan |
| `active_connections` | integer | Jumlah koneksi aktif saat ini |
| `is_active` | boolean | Status aktif user |
| `is_expired` | boolean | Apakah user sudah expired |
| `days_remaining` | integer | Jumlah hari tersisa (jika ada expiry) |
| `created_at` | string | Waktu user dibuat |
| `activated_at` | string | Waktu user diaktifkan |
| `expires_at` | string | Waktu expiry user |
| `last_login` | string/null | Waktu login terakhir |
| `notes` | string | Catatan tentang user |
| `valid_credentials` | boolean | Validasi kredensial (hanya muncul jika password diberikan) |

## Usage Examples

### 1. Cek User Tanpa Validasi Password

```bash
curl -X GET "http://localhost:8080/api/users/check/nizam" \
  -H "Cookie: admin-session=YOUR_SESSION_TOKEN"
```

### 2. Cek User Dengan Validasi Password

```bash
curl -X GET "http://localhost:8080/api/users/check/nizam?password=123" \
  -H "Cookie: admin-session=YOUR_SESSION_TOKEN"
```

### 3. JavaScript/Fetch Example

```javascript
// Cek user tanpa password
fetch('http://localhost:8080/api/users/check/nizam', {
  credentials: 'include'
})
.then(res => res.json())
.then(data => {
  if (data.code === 0) {
    console.log('User found:', data.data);
  } else {
    console.log('User not found');
  }
});

// Cek user dengan validasi password
fetch('http://localhost:8080/api/users/check/nizam?password=123', {
  credentials: 'include'
})
.then(res => res.json())
.then(data => {
  if (data.code === 0 && data.data.valid_credentials) {
    console.log('Valid credentials');
  } else {
    console.log('Invalid credentials');
  }
});
```

## Use Cases

1. **Validasi Login**: Cek apakah username dan password valid
2. **Cek Ketersediaan Username**: Verifikasi apakah username sudah digunakan
3. **Status User**: Melihat status aktif, expired, dan koneksi user
4. **Monitoring**: Melihat jumlah koneksi aktif user real-time

## Notes

- API ini memerlukan autentikasi admin
- Password di-hash menggunakan MD5 sebelum dibandingkan
- Active connections dihitung dari tabel `user_connections`
- Days remaining dihitung dari `expires_at` minus waktu sekarang
- User dianggap expired jika `expires_at < now()`
