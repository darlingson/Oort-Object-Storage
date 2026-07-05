# Signed URLs — Implementation Plan

## Concept
A signed URL is a time-limited, single-purpose token that acts as the sole credential. No JWT, no API key needed at the point of use.

| Mode | HTTP | Token lifecycle |
|------|------|----------------|
| Upload | `PUT /buckets/{bucket}/objects/{key}?token=...` | One-time use, deleted after upload |
| Download | `GET /buckets/{bucket}/objects/{key}?token=...` | Multiple uses until expiry |

## Flow

```
KYC service (API key or JWT)
  │
  │  POST /buckets/kyc-docs/objects/customer-789/passport.pdf/sign
  │  { "operation": "upload", "expires_in": "1h" }
  │
  ▼
{ "url": "http://.../buckets/kyc-docs/objects/customer-789/passport.pdf?token=oort_8f3a2b...",
  "expires_at": "2026-07-04T21:00:00Z" }
  │
  ▼ (KYC sends URL to customer)

Customer (no auth):
  PUT /buckets/kyc-docs/objects/customer-789/passport.pdf?token=oort_8f3a2b...
  --file→ passport scan

Server:
  1. Look up token in DB
  2. Verify: exists, not expired, operation = upload
  3. Verify: token's bucket+key match URL's bucket+key
  4. Call ObjectService.UploadObject (same as any authenticated upload)
  5. Delete token
  6. Return 201
```

## Files to create

| File | Purpose |
|------|---------|
| `migrations/0006_signed_urls.sql` | `signed_urls` table |
| `internal/storage/models/signed_url.go` | `SignedURL` struct |
| `internal/storage/repositories/signed_url_repository.go` | Interface |
| `internal/storage/repositories/postgres_signed_url_repository.go` | Postgres impl |
| `internal/services/signed_url_service.go` | Generate, Validate, Consume |
| `internal/api/handlers/signed_url_handler.go` | Sign, UploadWithToken, DownloadWithToken |

## Files to modify

| File | Change |
|------|--------|
| `internal/storage/models/permission.go` | Add `object:sign` |
| `internal/api/middleware/object_access.go` | Add `?token=` check as third auth path |
| `internal/api/routes/routes.go` | Add 3 routes |
| `cmd/server/main.go` | Wire new deps |

## Routes

| Method | Path | Auth | Permission |
|--------|------|------|------------|
| POST | `/buckets/{bucket}/objects/{key}/sign` | API key (scope) or JWT | `object:sign` |
| PUT | `/buckets/{bucket}/objects/{key}` with `?token=` | Token (open) | — |
| GET | `/buckets/{bucket}/objects/{key}` with `?token=` | Token (open) | — |

The `?token=` check goes first in `ObjectAccess` — if present, validate and skip API key / JWT checks.

## DB schema

```sql
CREATE TABLE signed_urls (
    id UUID PRIMARY KEY,
    bucket_name TEXT NOT NULL,
    object_key TEXT NOT NULL,
    token TEXT UNIQUE NOT NULL,
    operation TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
```

## Token format
`oort_` + 32 random bytes → base64 URL-safe (same pattern as API keys).
