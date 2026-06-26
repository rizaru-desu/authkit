# API Reference — MNS Backend

Base URL: `http://localhost:8080`

- Endpoint auth (Better Auth-compatible): prefix **`/api/auth`**
- Endpoint aplikasi: prefix **`/api/v1`**

## Autentikasi

Kirim salah satu:
- **Cookie** `mns_session` (otomatis di browser setelah sign-in), atau
- Header `Authorization: Bearer <token>` (mobile/API).

## Format Response

Ada **dua gaya** response:

**1. Better Auth style (raw, camelCase)** — dipakai endpoint `sign-*`, `get-session`, session self-service, dan `admin/*`:
```jsonc
// sukses → objek langsung
{ "token": "...", "user": { ... } }
// error
{ "message": "invalid email or password", "code": "INVALID_EMAIL_OR_PASSWORD" }
```

**2. Envelope style** — dipakai endpoint verification, two-factor, dan `/api/v1/*`:
```jsonc
// sukses
{ "success": true, "message": "email verified", "data": { ... } }
// error
{ "success": false, "error": "pesan error" }
```

## Objek umum

**User** (camelCase, dipakai endpoint Better Auth-style):
```json
{
  "id": "yxnoogedf31bfgf5ivtmrb3w",
  "email": "john@example.com",
  "name": "John",
  "image": null,
  "emailVerified": false,
  "role": "user",
  "twoFactorEnabled": false,
  "banned": false,
  "banReason": null,
  "banExpires": null,
  "createdAt": "2026-06-27T10:00:00Z",
  "updatedAt": "2026-06-27T10:00:00Z"
}
```

**Session** (camelCase):
```json
{
  "id": "sess_abc",
  "token": "8f3a...",
  "userId": "yxnoogedf31bfgf5ivtmrb3w",
  "expiresAt": "2026-07-04T10:00:00Z",
  "ipAddress": "127.0.0.1",
  "userAgent": "Mozilla/5.0",
  "impersonatedBy": null,
  "createdAt": "2026-06-27T10:00:00Z"
}
```

---

# 🔑 Auth — `/api/auth`

## POST `/sign-up/email`
Daftar user baru. **Tidak** auto sign-in secara default — kirim `autoSignIn: true` untuk langsung mendapat sesi.

**Request**
```json
{ "name": "John", "email": "john@example.com", "password": "password1234", "image": null, "autoSignIn": false }
```
| Field | Tipe | Wajib | Aturan |
|---|---|---|---|
| name | string | ✅ | |
| email | string | ✅ | format email |
| password | string | ✅ | 8–128 char |
| image | string\|null | — | |
| autoSignIn | boolean | — | default `false`; `true` → langsung dibuatkan sesi + cookie |

**Response 200**
```json
{ "token": "8f3a...", "user": { /* User */ } }
```
> `token` = `null` (user dibuat tanpa sesi) bila `autoSignIn` tidak `true`, **atau** bila `REQUIRE_EMAIL_VERIFICATION=true` (email verifikasi dikirim). Saat `token` `null`, arahkan user ke `/sign-in/email`.

**Error**: `400 BAD_REQUEST`, `422 USER_ALREADY_EXISTS`

---

## POST `/sign-in/email`
**Request**
```json
{ "email": "john@example.com", "password": "password1234", "code": "", "rememberMe": true }
```
| Field | Tipe | Wajib | Keterangan |
|---|---|---|---|
| email | string | ✅ | |
| password | string | ✅ | |
| code | string | — | kode 2FA (jika aktif) |
| rememberMe | boolean | — | default `true`; `false` → cookie sesi (hilang saat browser tutup) |

**Response 200**
```json
{ "redirect": false, "token": "8f3a...", "user": { /* User */ } }
```

**Error**
| Status | code | Kondisi |
|---|---|---|
| 401 | `INVALID_EMAIL_OR_PASSWORD` | kredensial salah |
| 401 | `TWO_FACTOR_REQUIRED` | 2FA aktif, `code` kosong (+ `"twoFactorRequired": true`) |
| 401 | `INVALID_TWO_FACTOR_CODE` | kode 2FA salah |
| 403 | `BANNED_USER` | user dibanned |
| 403 | `EMAIL_NOT_VERIFIED` | email belum diverifikasi (jika diwajibkan) |

---

## POST `/sign-out` 🔒
**Response 200**: `{ "success": true }`

## GET `/get-session` 🔒
**Response 200**
```json
{ "user": { /* User */ }, "session": { "id": "sess_abc", "expiresAt": "2026-07-04T10:00:00Z" } }
```

---

## Self-service Session (🔒)

### GET `/list-sessions`
**Response 200**: `{ "sessions": [ /* Session */ ] }`

### POST `/revoke-session`
**Request**: `{ "token": "8f3a..." }`
**Response 200**: `{ "success": true }` · **Error**: `403 FORBIDDEN` (bukan session milikmu), `404 NOT_FOUND`

### POST `/revoke-sessions`
Cabut **semua** session sendiri (termasuk yang aktif). **Response**: `{ "success": true }`

### POST `/revoke-other-sessions`
Cabut semua **kecuali** yang sekarang. **Response**: `{ "success": true }`

---

## Email Verification & Password Reset *(envelope style)*

### POST `/send-verification-email`
**Request**: `{ "email": "john@example.com" }`
**Response 200**: `{ "success": true, "message": "verification email sent" }`
> Selalu respons generik (anti user-enumeration).

### POST `/verify-email`
**Request**: `{ "token": "<token-dari-email>" }`
**Response 200**: `{ "success": true, "message": "email verified" }`
**Error**: `{ "success": false, "error": "invalid or expired code" }`

### POST `/request-password-reset`
**Request**: `{ "email": "john@example.com" }`
**Response 200**: `{ "success": true, "message": "password reset email sent" }`

### POST `/reset-password`
**Request**: `{ "token": "<token>", "password": "newpassword123" }` (password min 8)
**Response 200**: `{ "success": true, "message": "password updated" }`
> Semua session user dicabut setelah reset.

---

# 🔐 Two-Factor — `/api/auth/two-factor` (🔒, envelope style)

### POST `/enable`
**Request**: `{ "password": "password1234" }`
**Response 200**
```json
{
  "success": true,
  "message": "scan the QR / enter the URI in your authenticator, then verify",
  "data": {
    "totp_uri": "otpauth://totp/mns-backend:john@example.com?secret=...",
    "backup_codes": ["a1b2c3d4e5", "..."]
  }
}
```

### POST `/verify-totp`
Konfirmasi enrolment / aktifkan 2FA. **Request**: `{ "code": "123456" }`
**Response 200**: `{ "success": true, "message": "two-factor authentication enabled" }`

### POST `/disable`
**Request**: `{ "password": "password1234" }`
**Response 200**: `{ "success": true, "message": "two-factor authentication disabled" }`

---

# 👮 Admin — `/api/auth/admin` (🔒 + permission)

Tiap endpoint butuh permission `resource:action`. Default hanya `admin` yang punya semua (lihat RBAC di README).

### POST `/create-user` — `user:create`
**Request**: `{ "email": "u@x.com", "password": "password1234", "name": "User", "role": "user" }`
**Response 200**: `{ "user": { /* User */ } }` · **Error**: `422 USER_ALREADY_EXISTS`

### GET `/list-users` — `user:list`
**Query**: `searchValue` (cari di email/name), `limit` (default 100), `offset` (default 0)
**Response 200**
```json
{ "users": [ /* User */ ], "total": 42, "limit": 100, "offset": 0 }
```

### POST `/set-role` — `user:set-role`
**Request**: `{ "userId": "...", "role": "admin" }` → **Response**: `{ "user": { /* User */ } }`

### POST `/set-user-password` — `user:set-password`
**Request**: `{ "userId": "...", "newPassword": "newpassword123" }` → `{ "success": true }`
> Membuat credential account jika user belum punya.

### POST `/ban-user` — `user:ban`
**Request**
```json
{ "userId": "...", "banReason": "Spamming", "banExpiresIn": 604800 }
```
`banExpiresIn` = detik sampai ban berakhir (0/absen = permanen). Ban mencabut semua session user.
**Response**: `{ "user": { /* User, banned:true */ } }`

### POST `/unban-user` — `user:ban`
**Request**: `{ "userId": "..." }` → `{ "user": { /* User */ } }`

### POST `/list-user-sessions` — `session:list`
**Request**: `{ "userId": "..." }` → `{ "sessions": [ /* Session */ ] }`

### POST `/revoke-user-session` — `session:revoke`
**Request**: `{ "sessionToken": "..." }` → `{ "success": true }`

### POST `/revoke-user-sessions` — `session:revoke`
**Request**: `{ "userId": "..." }` → `{ "success": true }`

### POST `/impersonate-user` — `user:impersonate`
Buat session impersonasi (berlaku 1 jam). **Request**: `{ "userId": "..." }`
**Response 200**
```json
{ "token": "imp_...", "session": { /* Session, impersonatedBy: <adminId> */ }, "user": { /* User target */ } }
```

### POST `/remove-user` — `user:delete`
Hard delete (session & account cascade). **Request**: `{ "userId": "..." }` → `{ "success": true }`

### POST `/has-permission` 🔒
Cek permission. Tanpa `userId`/`role` → cek diri sendiri. Dengan `userId`/`role` → butuh caller admin.
**Request**
```json
{ "role": "moderator", "permissions": { "user": ["ban"] } }
```
**Response 200**: `{ "success": true }` (atau `false`)

### POST `/stop-impersonating` 🔒
Akhiri impersonasi (hapus session impersonasi + clear cookie). **Response**: `{ "success": true }`

---

# 🗂️ Aplikasi — `/api/v1` *(envelope style)*

### GET `/health`
**Response 200**: `{ "status": "ok" }`

### GET `/users` 🔒
**Query**: `limit` (default 20), `offset` (default 0)
**Response 200**
```json
{
  "success": true,
  "message": "ok",
  "data": [
    { "id": "...", "name": "John", "email": "john@example.com",
      "email_verified": false, "image": null, "role": "user", "created_at": "2026-06-27T10:00:00Z" }
  ]
}
```
> Catatan: `data` di sini **snake_case** (berbeda dari objek User camelCase di endpoint Better Auth-style).

### GET `/users/:id` 🔒
**Response 200**: `{ "success": true, "message": "ok", "data": { /* user snake_case */ } }`

### DELETE `/users/:id` 🔒
**Response 200**: `{ "success": true, "message": "user deleted" }`

---

# Ringkasan kode error

| code | Status | Arti |
|---|---|---|
| `BAD_REQUEST` | 400 | Body/validasi tidak valid |
| `UNAUTHORIZED` | 401 | Belum login / token invalid |
| `INVALID_EMAIL_OR_PASSWORD` | 401 | Kredensial salah |
| `TWO_FACTOR_REQUIRED` | 401 | Butuh kode 2FA |
| `INVALID_TWO_FACTOR_CODE` | 401 | Kode 2FA salah |
| `EMAIL_NOT_VERIFIED` | 403 | Email belum diverifikasi |
| `BANNED_USER` | 403 | User dibanned |
| `FORBIDDEN` | 403 | Permission kurang |
| `NOT_FOUND` | 404 | Resource tidak ada |
| `USER_ALREADY_EXISTS` | 422 | Email sudah terpakai |
| `INTERNAL_ERROR` | 500 | Error server |
