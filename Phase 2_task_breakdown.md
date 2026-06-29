# Oort — Developer Experience

## Phase 2 Goal

At the end of Phase 2, this should work:

```bash
# Create API key
curl -X POST \
  /api-keys

# Upload using API key
curl -H "Authorization: Bearer oort_xxxxx" \
  -X PUT \
  /buckets/documents/objects/file.pdf

# Generate signed URL
curl -X POST \
  /buckets/documents/objects/file.pdf/sign

# Download using signed URL
curl \
  "http://localhost:3333/download/abc123..."
```

And:

* API keys authenticate requests
* Public signed URLs work without authentication
* URLs expire automatically
* Responses are consistent JSON
* Health endpoints exist
* CLI can interact with the server

---

## Dependency Graph

```text
Database
    |
    v
API Key Model
    |
    v
API Key Repository
    |
    v
API Key Service
    |
    v
Authentication Middleware
    |
    +---------------------------+
    |                           |
    v                           |
Signed URL Model               |
    |                           |
    v                           |
Signed URL Repository          |
    |                           |
    v                           |
Signed URL Service             |
    |                           |
    v                           |
Signed URL API                 |
    |                           |
    +---------------------------+
    |
    v
Standard API Responses
    |
    v
Health Endpoints
    |
    v
CLI Client
```

---

# Epic 1 — API Keys

Everything else depends on authentication.

Without API keys:

* Anyone can upload
* Anyone can delete
* No access boundaries

---

## Task 1.1 — API Key Model

**File:**

```text
internal/storage/models/api_key.go
```

```go
type APIKey struct {
    ID          uuid.UUID
    Name        string
    KeyHash     string
    Buckets     []string // bucket allowlist; ["*"] = full access
    CreatedAt   time.Time
    LastUsedAt  *time.Time
}
```

The `Buckets` field is the isolation mechanism. Examples:

| Key Name        | Buckets              | Can Access                     |
|-----------------|----------------------|--------------------------------|
| contracts-svc   | ["contracts"]        | contracts bucket only          |
| kyc-svc         | ["kyc-uploads"]      | kyc-uploads bucket only        |
| admin           | ["*"]                | everything                     |

---

## Task 1.2 — API Key Repository

Responsibilities:

* Create key (with bucket allowlist)
* Find by hash
* List keys
* Delete key

Methods:

```go
Create(ctx, key *APIKey) error
FindByHash(ctx, hash string) (*APIKey, error)
List(ctx) ([]APIKey, error)
Delete(ctx, id uuid.UUID) error
```

**Migration:**

```sql
CREATE TABLE api_keys (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    key_hash TEXT UNIQUE NOT NULL,
    buckets TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    last_used_at TIMESTAMP
);
```

Store `Buckets` as a Postgres text array. `{"*"}` means unrestricted.

---

## Task 1.3 — API Key Service

Business Rules:

* Generate secure keys (`oort_` prefix + 32 random bytes -> base64)
* Never store raw key
* Store SHA256 hash only
* Return raw key once on creation
* Validate bucket access: given a key and a bucket name, does the key's allowlist permit it?

**Key method:**

```go
func (s *KeyService) CanAccessBucket(key *APIKey, bucketName string) bool {
    for _, allowed := range key.Buckets {
        if allowed == "*" || allowed == bucketName {
            return true
        }
    }
    return false
}
```

Example:

```text
oort_f3b2f8...
```

---

## Task 1.4 — API Key API

Endpoints:

| Method | Endpoint         | Description                            |
| ------ | ---------------- | -------------------------------------- |
| POST   | /api-keys        | Create key (body: `{name, buckets}`)   |
| GET    | /api-keys        | List keys (never expose key_hash)      |
| DELETE | /api-keys/{id}   | Delete key                             |

**POST body example:**

```json
{
  "name": "kyc-service",
  "buckets": ["kyc-uploads"]
}
```

**POST response (raw key shown once):**

```json
{
  "id": "uuid...",
  "name": "kyc-service",
  "buckets": ["kyc-uploads"],
  "raw_key": "oort_f3b2f8...",
  "created_at": "..."
}
```

---

## Task 1.5 — API Key Tests

Verify:

* Key creation returns raw key once
* Hash stored, raw key not retrievable via GET
* Bucket allowlist stored and returned
* Wildcard `["*"]` permits all buckets
* Delete works
* Duplicate name allowed (keys identified by ID)

---

# Epic 2 — Authentication Middleware

Protect write operations **and enforce bucket scoping**.

---

## Task 2.1 — Authentication Middleware

Responsibilities:

* Read Authorization header
* Validate API key (find by hash)
* Attach resolved key to request context
* Middleware runs on **all** routes (even public ones can skip by not passing a key)

**Context key:**

```go
type contextKey string
const KeyContextKey = contextKey("api_key")

func GetAPIKey(ctx context.Context) *models.APIKey {
    val := ctx.Value(KeyContextKey)
    if val == nil {
        return nil
    }
    return val.(*models.APIKey)
}
```

Example:

```http
Authorization: Bearer oort_xxx
```

---

## Task 2.2 — Bucket Access Enforcer

Separate middleware that runs after authentication **on bucket-scoped routes**.

Responsibilities:

* Extract `{bucket}` from the URL path
* Get the API key from context
* Call `CanAccessBucket(key, bucketName)`
* Reject with 403 if the key's allowlist doesn't include the bucket

**Logic:**

```
Request: PUT /buckets/contracts/objects/report.pdf
    |
    v
Authenticate -> key found (contracts-svc)
    |
    v
Enforce Scope -> key.Buckets = ["contracts"]
                  bucket in path = "contracts"
                  "contracts" in ["contracts"] -> allow
    |
    v
Handler runs

---

Request: PUT /buckets/kyc-uploads/objects/photo.jpg
    |
    v
Authenticate -> same key (contracts-svc)
    |
    v
Enforce Scope -> key.Buckets = ["contracts"]
                  bucket in path = "kyc-uploads"
                  "kyc-uploads" not in ["contracts"] -> 403 Forbidden
```

---

## Task 2.3 — Route Protection

Require authentication for:

```text
PUT object
DELETE object
POST bucket
DELETE bucket (future)
```

Allow (no auth required):

```text
GET health
GET signed url downloads
```

Bucket access enforcer runs on all authenticated bucket-scoped routes.

---

## Task 2.4 — Middleware Tests

Verify:

* Missing key rejected (401)
* Invalid key rejected (401)
* Valid key accepted (200)
* Key with ["contracts"] can access contracts bucket
* Key with ["contracts"] cannot access kyc-uploads bucket (403)
* Wildcard key ["*"] can access any bucket
* Public routes work without key

---

# Epic 3 — Signed URLs

Temporary access without API key.

---

## Task 3.1 — Signed URL Model

```go
type SignedURL struct {
    ID          uuid.UUID
    BucketName  string
    ObjectKey   string
    Token       string
    ExpiresAt   time.Time
    CreatedAt   time.Time
}
```

---

## Task 3.2 — Signed URL Repository

Methods:

```go
Create(...)
Find(...)
Delete(...)
```

---

## Task 3.3 — Signed URL Service

Flow:

```text
Validate Object
      |
Generate Token
      |
Persist Token
      |
Return URL
```

---

## Task 3.4 — Signed URL API

Endpoints:

| Method | Endpoint                             |
| ------ | ------------------------------------ |
| POST   | /buckets/{bucket}/objects/{key}/sign |
| GET    | /download/{token}                    |

---

## Task 3.5 — Signed URL Tests

Verify:

* URL generated
* Download succeeds
* Expired URL rejected

---

# Epic 4 — Standard API Responses

Currently handlers return mixed responses.

Normalize everything.

---

## Task 4.1 — Response Models

Success:

```json
{
  "success": true,
  "data": {}
}
```

Error:

```json
{
  "success": false,
  "error": {
    "code": "object_not_found",
    "message": "Object not found"
  }
}
```

---

## Task 4.2 — Response Helpers

File:

```text
internal/api/responses
```

Functions:

```go
Success(...)
Created(...)
Error(...)
```

---

## Task 4.3 — Response Refactor

Update:

* Bucket handlers
* Object handlers
* Future endpoints

---

## Task 4.4 — Response Tests

Verify:

* Structure consistent
* Error codes present

---

# Epic 5 — Health Endpoints

For monitoring and operations.

---

## Task 5.1 — Liveness Endpoint

```http
GET /health/live
```

Response:

```json
{
  "status":"ok"
}
```

---

## Task 5.2 — Readiness Endpoint

Checks:

* PostgreSQL
* Storage directory

```http
GET /health/ready
```

---

## Task 5.3 — Health Tests

Verify:

* DB failure detected
* Storage failure detected

---

# Epic 6 — CLI Tool

Makes Oort usable without curl.

---

## Task 6.1 — CLI Project

```text
cmd/oortctl
```

Commands:

```bash
oortctl bucket create
oortctl bucket list
oortctl upload
oortctl download
```

---

## Task 6.2 — API Client

Reusable HTTP client.

```text
internal/client
```

---

## Task 6.3 — Bucket Commands

```bash
oortctl bucket create documents
oortctl bucket list
```

---

## Task 6.4 — Object Commands

```bash
oortctl upload contract.pdf
oortctl download contract.pdf
```

---

## Task 6.5 — CLI Tests

Verify:

* Command execution
* Request generation
* Error handling

---

# Actual Execution Order

Don't jump around.

## Sprint 1

* [ ] API key model (with Buckets allowlist)
* [ ] API key repository + migration
* [ ] API key service (key gen, hash, CanAccessBucket)
* [ ] API key API (POST/GET/DELETE, raw key on creation)
* [ ] API key tests

## Sprint 2

* [ ] Authentication middleware (validate key, attach to context)
* [ ] Bucket access enforcer middleware (CanAccessBucket check)
* [ ] Route protection config (public vs authenticated)
* [ ] Middleware + enforcer tests

## Sprint 3

* [ ] Signed URL model
* [ ] Signed URL repository
* [ ] Signed URL service
* [ ] Signed URL API
* [ ] Signed URL tests

## Sprint 4

* [ ] Response models
* [ ] Response helpers
* [ ] Handler refactor
* [ ] Response tests

## Sprint 5

* [ ] Liveness endpoint
* [ ] Readiness endpoint
* [ ] Health tests

## Sprint 6

* [ ] CLI project
* [ ] API client
* [ ] Bucket commands
* [ ] Object commands
* [ ] CLI tests