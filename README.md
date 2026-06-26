# MNS Backend

Backend autentikasi & user management bergaya **[Better Auth](https://www.better-auth.com/)**, ditulis dengan **Go** + **Clean Architecture**. Bisa dipakai sebagai *starter* aplikasi user management (admin panel, RBAC, 2FA, dll).

API-nya kompatibel dengan konvensi Better Auth (`/api/auth/*`, response `{ token, user }`), jadi bisa dikonsumsi langsung oleh `better-auth` client di frontend.

---

## ✨ Fitur

- **Auth email/password** — sign-up, sign-in, sign-out, get-session
- **Session** — opaque token tersimpan di DB; transport **httpOnly cookie** (web) **+ Bearer** (mobile/API)
- **Email verification & password reset** — token di-hash, dikirim via email (SMTP) atau di-log (dev)
- **Two-factor (2FA)** — TOTP + backup codes (sekali pakai)
- **Self-service session** — user kelola session sendiri (list / revoke / revoke-others)
- **Admin plugin** — list/search user, create, set-role, set-password, ban/unban, impersonate, remove, kelola session
- **RBAC / Access Control** — role + permission custom (resource→action), `has-permission`
- **Rate limit** (in-memory), **CORS + trusted origins** (proteksi CSRF)
- `requireEmailVerification` — blokir login sampai email terverifikasi

---

## 🧱 Tech Stack

| Komponen | Library |
|---|---|
| Bahasa | Go 1.25 |
| HTTP framework | [gin-gonic/gin](https://github.com/gin-gonic/gin) |
| Database | PostgreSQL via [jackc/pgx/v5](https://github.com/jackc/pgx) |
| Query (type-safe) | [sqlc](https://sqlc.dev) |
| Logger | [uber-go/zap](https://github.com/uber-go/zap) |
| ID | [cuid2](https://github.com/nrednav/cuid2) |
| 2FA | [pquerna/otp](https://github.com/pquerna/otp) |
| Password | bcrypt (`golang.org/x/crypto`) |

---

## 🏗️ Arsitektur

Clean Architecture — dependency mengarah ke dalam (domain tidak tahu soal HTTP/DB):

```
cmd/server                → entry point & dependency injection
internal/
  domain/
    entity/               → model domain (User, Session, Account, ...)
    repository/           → interface (kontrak data access)
  usecase/                → business logic (auth, admin, 2FA, verification)
  repository/postgres/    → implementasi repo (membungkus sqlc)
    query/                → file .sql (sumber kebenaran sqlc)
    sqlc/                 → kode generated (JANGAN diedit)
  delivery/http/
    handler/              → HTTP handlers
    middleware/           → auth, rate limit, CORS, permission
    router.go             → registrasi route
pkg/                      → util reusable (config, access, email, totp, secure, ...)
migrations/               → migrasi SQL (golang-migrate)
```

**Alur:** `handler → usecase → repository (interface) → postgres adapter → sqlc`

---

## 🚀 Quick Start

### Prasyarat
- Go 1.25+
- PostgreSQL 13+
- [`golang-migrate`](https://github.com/golang-migrate/migrate) (untuk migrasi)
- [`sqlc`](https://docs.sqlc.dev/en/latest/overview/install.html) (hanya jika mengubah query)

### Langkah
```bash
# 1. Konfigurasi
cp .env.example .env
#   wajib diisi: DATABASE_URL, JWT_SECRET

# 2. Migrasi database
migrate -path migrations -database "$DATABASE_URL" up

# 3. Jalankan server
make run            # http://localhost:8080

# 4. Buat admin pertama (manual) — promote user yang sudah sign-up:
#    UPDATE users SET role='admin' WHERE email='you@example.com';
```

### Perintah Makefile
| Perintah | Fungsi |
|---|---|
| `make run` | Jalankan server |
| `make build` | Build binary ke `bin/` |
| `make test` | Test + race detector |
| `make lint` | golangci-lint |
| `make sqlc` | Generate ulang kode dari `*.sql` |
| `make migrate-up` / `migrate-down` | Migrasi |

---

## ⚙️ Konfigurasi (Environment Variables)

| Variable | Default | Keterangan |
|---|---|---|
| `APP_NAME` | `mns-backend` | Nama app (issuer 2FA) |
| `APP_ENV` | `development` | `development` / `production` |
| `SERVER_HOST` / `SERVER_PORT` | `0.0.0.0` / `8080` | Alamat server |
| `DATABASE_URL` | — *(wajib)* | `postgres://user:pass@host:5432/db?sslmode=disable` |
| `DB_MAX_OPEN_CONNS` / `DB_MAX_IDLE_CONNS` | `25` / `5` | Pool koneksi |
| `JWT_SECRET` | — *(wajib)* | Secret (disimpan untuk kompatibilitas; session pakai opaque token) |
| `SESSION_EXPIRY` | `168h` | Masa berlaku session |
| `REQUIRE_EMAIL_VERIFICATION` | `false` | Blokir login sampai email terverifikasi |
| `SESSION_COOKIE_NAME` | `mns_session` | Nama cookie |
| `SESSION_COOKIE_SECURE` | `false` | `true` di production (HTTPS) |
| `SESSION_COOKIE_SAMESITE` | `lax` | `lax` / `strict` / `none` |
| `RATE_LIMIT_WINDOW` / `RATE_LIMIT_MAX` | `60s` / `100` | Rate limit per IP |
| `APP_BASE_URL` | `http://localhost:3000` | Base URL frontend (untuk link email) |
| `EMAIL_SMTP_HOST` | *(kosong)* | Kosong = log link ke console (mode dev) |
| `EMAIL_SMTP_PORT` | `587` | Port SMTP |
| `EMAIL_SMTP_USER` / `EMAIL_SMTP_PASS` | — | Kredensial SMTP |
| `EMAIL_FROM_ADDRESS` / `EMAIL_FROM_NAME` | `no-reply@example.com` / `mns-backend` | Pengirim |
| `TRUSTED_ORIGINS` | *(kosong)* | CORS/CSRF allowlist, koma. Wildcard `*` `**` `?` |

---

## 📡 API Reference

Base URL: `http://localhost:8080`

Endpoint auth ber-prefix **`/api/auth`** (kompatibel Better Auth). Endpoint aplikasi ber-prefix **`/api/v1`**.

Autentikasi: kirim **cookie** `mns_session` (otomatis di browser) **atau** header `Authorization: Bearer <token>`.

### 🔑 Auth — `/api/auth`

| Method | Path | Auth | Keterangan |
|---|---|---|---|
| POST | `/sign-up/email` | — | Daftar `{ name, email, password, image?, autoSignIn? }` → `{ token, user }` (`token` `null` kecuali `autoSignIn:true`) |
| POST | `/sign-in/email` | — | Login `{ email, password, code?, rememberMe? }` → `{ token, user }` |
| POST | `/sign-out` | ✅ | Logout |
| GET | `/get-session` | ✅ | `{ user, session }` |
| POST | `/send-verification-email` | — | Kirim email verifikasi `{ email }` |
| POST | `/verify-email` | — | Verifikasi `{ token }` |
| POST | `/request-password-reset` | — | Minta reset `{ email }` |
| POST | `/reset-password` | — | Reset `{ token, password }` |

### 🔐 Two-Factor — `/api/auth/two-factor` (semua ✅)

| Method | Path | Keterangan |
|---|---|---|
| POST | `/enable` | `{ password }` → `{ totpURI, backupCodes }` |
| POST | `/verify-totp` | `{ code }` → aktifkan 2FA |
| POST | `/disable` | `{ password }` |

> Jika 2FA aktif, sertakan `code` saat sign-in. Tanpa code → `401 TWO_FACTOR_REQUIRED`.

### 🪪 Self-service Session — `/api/auth` (semua ✅)

| Method | Path | Keterangan |
|---|---|---|
| GET | `/list-sessions` | Session milik user |
| POST | `/revoke-session` | Cabut 1 session `{ token }` |
| POST | `/revoke-sessions` | Cabut semua session sendiri |
| POST | `/revoke-other-sessions` | Cabut semua kecuali yang sekarang |

### 👮 Admin — `/api/auth/admin` (✅ + permission)

Setiap endpoint butuh permission `resource:action`. Default hanya role `admin` yang punya semua.

| Method | Path | Permission | Keterangan |
|---|---|---|---|
| POST | `/create-user` | `user:create` | `{ email, password, name, role? }` |
| GET | `/list-users` | `user:list` | `?searchValue=&limit=&offset=` → `{ users, total, limit, offset }` |
| POST | `/set-role` | `user:set-role` | `{ userId, role }` |
| POST | `/set-user-password` | `user:set-password` | `{ userId, newPassword }` |
| POST | `/ban-user` | `user:ban` | `{ userId, banReason?, banExpiresIn? }` |
| POST | `/unban-user` | `user:ban` | `{ userId }` |
| POST | `/list-user-sessions` | `session:list` | `{ userId }` |
| POST | `/revoke-user-session` | `session:revoke` | `{ sessionToken }` |
| POST | `/revoke-user-sessions` | `session:revoke` | `{ userId }` |
| POST | `/impersonate-user` | `user:impersonate` | `{ userId }` → session impersonasi (1 jam) |
| POST | `/remove-user` | `user:delete` | `{ userId }` (hard delete) |
| POST | `/has-permission` | ✅ (auth) | `{ userId?, role?, permissions }` → `{ success }` |
| POST | `/stop-impersonating` | ✅ (auth) | Hentikan impersonasi |

### 🗂️ Aplikasi — `/api/v1`

| Method | Path | Auth | Keterangan |
|---|---|---|---|
| GET | `/health` | — | Health check |
| GET | `/users` | ✅ | List user (admin/internal) |
| GET | `/users/:id` | ✅ | Detail user |
| DELETE | `/users/:id` | ✅ | Hapus user |

---

## 🔒 Access Control (RBAC)

Role & permission didefinisikan di `pkg/access/access.go` (`DefaultController`):

| Role | Permission |
|---|---|
| `admin` | semua (`user:*`, `session:*`) |
| `moderator` | `user:[list,ban]`, `session:[list,revoke]` |
| `user` | — (tidak ada akses admin) |

Tambah role custom:
```go
New(statement).
    SetRole("editor", Role{
        ResourceUser: {ActionList, ActionSetPassword},
    })
```
Set role user via `POST /api/auth/admin/set-role`.

---

## 🧪 Contoh Penggunaan (curl)

```bash
# Sign-up tanpa auto sign-in (default) → { token: null, user }, tidak ada cookie
curl -X POST http://localhost:8080/api/auth/sign-up/email \
  -H "Content-Type: application/json" \
  -d '{"name":"John","email":"john@example.com","password":"password1234"}'

# Sign-up + langsung login (autoSignIn:true) → set cookie sesi
curl -X POST http://localhost:8080/api/auth/sign-up/email \
  -H "Content-Type: application/json" -c cookies.txt \
  -d '{"name":"John","email":"john@example.com","password":"password1234","autoSignIn":true}'

# Sign-in
curl -X POST http://localhost:8080/api/auth/sign-in/email \
  -H "Content-Type: application/json" -c cookies.txt \
  -d '{"email":"john@example.com","password":"password1234"}'

# Get session (pakai cookie)
curl http://localhost:8080/api/auth/get-session -b cookies.txt

# Admin: list users (butuh role admin)
curl "http://localhost:8080/api/auth/admin/list-users?limit=20" -b cookies.txt
```

### Konsumsi dari frontend (Better Auth client)
```ts
import { createAuthClient } from "better-auth/client";

export const authClient = createAuthClient({
  baseURL: "http://localhost:8080",            // basePath /api/auth otomatis
  fetchOptions: { credentials: "include" },     // kirim cookie
});

await authClient.signIn.email({ email, password });

// Sign-up default backend ini TIDAK auto sign-in. Kirim autoSignIn:true
// jika ingin langsung mendapat sesi (mirip default Better Auth).
await authClient.signUp.email({ name, email, password, autoSignIn: true });
```
> Catatan: skema DB pakai `snake_case`, jadi map field di config Better Auth frontend (`emailVerified → email_verified`, dst).
>
> Default `autoSignIn` backend ini = `false` (beda dari default Better Auth yang `true`). Tanpa `autoSignIn:true`, response `token` = `null` dan user harus sign-in terpisah.

---

## 🗃️ Skema Database

4 tabel inti Better Auth + ekstensi plugin:

- **`users`** — id, name, email, email_verified, image, role, two_factor_enabled, banned, ban_reason, ban_expires
- **`sessions`** — id, user_id, token, expires_at, ip_address, user_agent, impersonated_by
- **`accounts`** — kredensial & OAuth (password tersimpan di sini, `provider_id='credential'`)
- **`verifications`** — token email-verify / reset-password (di-hash)
- **`two_factors`** — secret TOTP + backup codes

---

## 🔐 Catatan Keamanan / Production

- Set `SESSION_COOKIE_SECURE=true` + HTTPS di production.
- Frontend beda origin → `SESSION_COOKIE_SAMESITE=none`, `TRUSTED_ORIGINS=<url-frontend>`, fetch `credentials:"include"`.
- Konfigurasi SMTP (`EMAIL_SMTP_HOST`) agar token verify/reset dikirim via email (bukan di-log).
- Rate limit saat ini in-memory (tidak shared antar instance) — ganti Redis untuk multi-instance.
- Password di-hash bcrypt; token verify/reset & backup code disimpan sebagai SHA-256.

## 🚧 Belum termasuk (roadmap)
- Social/OAuth login (Google, GitHub) — tabel `accounts` sudah siap
- Magic link / email OTP, organization, passkey
- Bootstrap admin otomatis, docker-compose, test suite, frontend

---

## 📁 Struktur Direktori
```
cmd/server/          # main.go
internal/
  domain/            # entity + repository interface
  usecase/           # business logic
  repository/postgres/  # adapter sqlc
  delivery/http/     # handler, middleware, router
pkg/                 # access, config, email, totp, secure, id, logger, ...
migrations/          # *.up.sql / *.down.sql
sqlc.yaml            # konfigurasi sqlc
Makefile
.env.example
```
