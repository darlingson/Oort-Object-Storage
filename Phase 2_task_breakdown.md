# Oort — Developer Experience (Phase 2)

## Phase 2 Goal

At the end of Phase 2, this should work:

```bash
# Login as admin
curl -X POST /auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@oort.local","password":"admin123"}'
# → { "token": "eyJhbGci..." }

# Create API key (requires JWT + apikey:create permission)
curl -X POST /api-keys \
  -H "Authorization: Bearer eyJhbGci..." \
  -H "Content-Type: application/json" \
  -d '{"name":"ci-service","buckets":["artifacts"]}'
# → { "raw_key": "oort_f3b2f8..." }

# Upload using JWT auth + API key for bucket scoping
curl -X PUT /buckets/artifacts/objects/release.tar.gz \
  -H "Authorization: Bearer eyJhbGci..." \
  -H "X-API-Key: oort_f3b2f8..." \
  -F "file=@release.tar.gz"

# Or upload with just JWT (if user has direct bucket access)
curl -X PUT /buckets/artifacts/objects/release.tar.gz \
  -H "Authorization: Bearer eyJhbGci..." \
  -F "file=@release.tar.gz"

# Generate signed URL
curl -X POST /buckets/documents/objects/file.pdf/sign \
  -H "Authorization: Bearer eyJhbGci..."

# Download using signed URL (no auth)
curl "http://localhost:3333/download/abc123..."
```

And:

* JWT tokens authenticate requests (who you are)
* API keys control bucket scoping (which buckets you can touch)
* Permissions control what operations you can perform
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
    +---------------------------+
    |                           |
    v                           v
User Model                 API Key Model
    |                           |
    v                           v
User Repository             API Key Repository
    |                           |
    v                           v
User Service                API Key Service
    |                           |
    v                           v
JWT Service                 Bucket Scope Middleware
    |                           |
    v                           v
Auth Middleware              (uses X-API-Key header)
    |                           |
    v                           v
+--- Permission Middleware ---+
|   (checks role permissions)  |
+------------------------------+
    |
    v
Signed URLs
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

# Epic 1 — API Keys (Scope Controller)

API keys no longer authenticate. They control **which buckets** a request can access.

Without API keys:

* JWT-authenticated users access buckets based on their permissions
* No per-integration bucket isolation

With API keys:

* Integrations get scoped to specific buckets
* Keys act as a bucket allowlist filter on top of JWT auth
* A request can include both `Authorization: Bearer <JWT>` and `X-API-Key: <key>`

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
| ci-deploy       | ["*"]                | everything                     |

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
  "name": "ci-service",
  "buckets": ["artifacts"]
}
```

**POST response (raw key shown once):**

```json
{
  "id": "uuid...",
  "name": "ci-service",
  "buckets": ["artifacts"],
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

# Epic 2 — JWT Authentication

Authenticate requests and identify the user.

---

## Task 2.1 — User Model

```go
type User struct {
    ID           uuid.UUID
    Email        string
    PasswordHash string
    CreatedAt    time.Time
}
```

**Migration:**

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
```

---

## Task 2.2 — User Repository

Methods:

```go
Create(ctx, user *User) error
FindByEmail(ctx, email string) (*User, error)
FindByID(ctx, id uuid.UUID) (*User, error)
```

---

## Task 2.3 — User Service

Business Rules:

* Register: validate email, hash password with bcrypt, store
* Login: find by email, verify bcrypt, return JWT
* Get user by ID (for context identity)

---

## Task 2.4 — JWT Service

Business Rules:

* Sign: create JWT with user_id, email, roles, permissions, expiry claims
* Validate: parse token, verify signature, check expiry
* Extract claims: return user identity from token

```go
type Claims struct {
    UserID      string   `json:"user_id"`
    Email       string   `json:"email"`
    Roles       []string `json:"roles"`
    Permissions []string `json:"permissions"`
    jwt.RegisteredClaims
}
```

---

## Task 2.5 — Auth Handler

Endpoints:

| Method | Endpoint      | Description              |
| ------ | ------------- | ------------------------ |
| POST   | /auth/login   | Login, returns JWT       |

**POST body:**

```json
{
  "email": "admin@oort.local",
  "password": "admin123"
}
```

**POST response:**

```json
{
  "token": "eyJhbGci...",
  "user": {
    "id": "uuid...",
    "email": "admin@oort.local"
  }
}
```

---

## Task 2.6 — Auth Middleware

Responsibilities:

* Read `Authorization: Bearer <token>` header
* Validate JWT
* Attach resolved user + permissions to request context

**Context key:**

```go
type contextKey string

const UserContextKey = contextKey("user")
const PermissionsContextKey = contextKey("permissions")

func GetUser(ctx context.Context) *models.User { ... }
func GetPermissions(ctx context.Context) []string { ... }
```

---

## Task 2.7 — First-Run Admin Seeding

In `main.go`, after migrations:

1. Check if any users exist
2. If not, create admin user from `ADMIN_EMAIL` / `ADMIN_PASSWORD` env vars
3. Assign admin role with all permissions

Config additions:

```env
JWT_SECRET=your-secret-key-here
ADMIN_EMAIL=admin@oort.local
ADMIN_PASSWORD=admin123
```

---

## Task 2.8 — Auth Tests

Verify:

* Login with valid credentials returns JWT
* Login with invalid password rejected (401)
* JWT validates correctly
* Expired JWT rejected (401)
* Missing auth header rejected (401)
* Middleware attaches user to context

---

# Epic 3 — Permissions / RBAC

Control what operations each user can perform.

---

## Task 3.1 — Permission Constants

```go
const (
    BucketCreate = "bucket:create"
    BucketList   = "bucket:list"
    BucketDelete = "bucket:delete"

    ObjectUpload   = "object:upload"
    ObjectDownload = "object:download"
    ObjectDelete   = "object:delete"
    ObjectList     = "object:list"

    APIKeyCreate = "apikey:create"
    APIKeyList   = "apikey:list"
    APIKeyDelete = "apikey:delete"
)
```

---

## Task 3.2 — Role Model

```go
type Role struct {
    ID          uuid.UUID
    Name        string
    Permissions []string
    CreatedAt   time.Time
}
```

**Migration:**

```sql
CREATE TABLE roles (
    id UUID PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    permissions TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE user_roles (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);
```

---

## Task 3.3 — Role Repository

Methods:

```go
Create(ctx, role *Role) error
FindByName(ctx, name string) (*Role, error)
FindByUserID(ctx, userID uuid.UUID) ([]Role, error)
AssignToUser(ctx, userID, roleID uuid.UUID) error
```

---

## Task 3.4 — Permission Middleware

```go
func RequirePermission(permission string) func(http.Handler) http.Handler
```

Responsibilities:

* Get permissions from request context (set by auth middleware)
* Check if the required permission is in the user's permissions
* Reject with 403 if not present

**Route-permission mapping:**

| Route                                      | Required Permission  |
|--------------------------------------------|----------------------|
| POST /buckets                              | bucket:create        |
| GET /buckets                               | bucket:list          |
| GET /buckets/{name}                        | bucket:list          |
| PUT /buckets/{bucket}/objects/*            | object:upload        |
| GET /buckets/{bucket}/objects/*            | object:download      |
| DELETE /buckets/{bucket}/objects/*         | object:delete        |
| GET /buckets/{bucket}/objects/             | object:list          |
| POST /api-keys                             | apikey:create        |
| GET /api-keys                              | apikey:list          |
| DELETE /api-keys/{id}                      | apikey:delete        |

---

## Task 3.5 — Permission Tests

Verify:

* User with permission can access route
* User without permission gets 403
* Multiple permissions work
* Admin role has all permissions

---

# Epic 4 — Bucket Scope Middleware

Refactor the existing `BucketScope` middleware to be a pure scope controller (no auth).

---

## Task 4.1 — Refactor Scope Middleware

Current `scope.go` does both auth + bucket-scoping. Split into:

1. **Auth middleware** — JWT validation (Epic 2)
2. **BucketScope middleware** — reads `X-API-Key`, validates, checks bucket access

The new BucketScope middleware:

* Reads `X-API-Key` header (optional)
* If present, validates the key and checks `CanAccessBucket`
* If absent, skips scope check (JWT permissions alone are sufficient)
* If key present but bucket not allowed, rejects with 403

---

## Task 4.2 — Route Protection

Final middleware stack per route group:

```text
/buckets/{bucket}/objects/* -> Auth -> RequirePermission -> BucketScope -> Handler
/buckets/*                  -> Auth -> RequirePermission -> Handler
/api-keys/*                 -> Auth -> RequirePermission -> Handler
/auth/*                     -> (no middleware)
/health                     -> (no middleware)
```

---

## Task 4.3 — Scope Tests (Updated)

Verify:

* Valid API key + correct bucket passes scope check
* Valid API key + wrong bucket rejected (403)
* Invalid API key rejected (403)
* Expired API key rejected (403)
* Missing API key (with valid JWT) passes scope check (scope is optional)
* Wildcard key works

---

# Epic 5 — Signed URLs

Temporary access without authentication.

(Unchanged from original — no pivot here)

---

## Task 5.1 — Signed URL Model

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

## Task 5.2 — Signed URL Repository

Methods:

```go
Create(...)
Find(...)
Delete(...)
```

---

## Task 5.3 — Signed URL Service

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

## Task 5.4 — Signed URL API

Endpoints:

| Method | Endpoint                             |
| ------ | ------------------------------------ |
| POST   | /buckets/{bucket}/objects/{key}/sign |
| GET    | /download/{token}                    |

---

## Task 5.5 — Signed URL Tests

Verify:

* URL generated
* Download succeeds
* Expired URL rejected

---

# Epic 6 — Standard API Responses

Normalize all handler responses.

---

## Task 6.1 — Response Models

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

## Task 6.2 — Response Helpers

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

## Task 6.3 — Response Refactor

Update:

* Bucket handlers
* Object handlers
* Auth handlers
* Key handlers
* Health handlers

---

## Task 6.4 — Response Tests

Verify:

* Structure consistent
* Error codes present

---

# Epic 7 — Health Endpoints

For monitoring and operations.

---

## Task 7.1 — Liveness Endpoint

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

## Task 7.2 — Readiness Endpoint

Checks:

* PostgreSQL
* Storage directory

```http
GET /health/ready
```

---

## Task 7.3 — Health Tests

Verify:

* DB failure detected
* Storage failure detected

---

# Epic 8 — CLI Tool

Makes Oort usable without curl.

---

## Task 8.1 — CLI Project

```text
cmd/oortctl
```

Commands:

```bash
oortctl login
oortctl bucket create
oortctl bucket list
oortctl upload
oortctl download
```

---

## Task 8.2 — API Client

Reusable HTTP client.

```text
internal/client
```

---

## Task 8.3 — Auth Commands

```bash
oortctl login
```

Stores JWT token locally for subsequent commands.

---

## Task 8.4 — Bucket Commands

```bash
oortctl bucket create documents
oortctl bucket list
```

---

## Task 8.5 — Object Commands

```bash
oortctl upload contract.pdf
oortctl download contract.pdf
```

---

## Task 8.6 — CLI Tests

Verify:

* Command execution
* Request generation
* Token storage
* Error handling

---

# Actual Execution Order

## Sprint 1 ✅ (Completed — original API key implementation)

* [x] API key model (with Buckets allowlist)
* [x] API key repository + migration
* [x] API key service (key gen, hash, CanAccessBucket)
* [x] API key API (POST/GET/DELETE, raw key on creation)
* [x] API key tests

## Sprint 2 ✅ (Completed — new architecture)

* [x] Bucket scope middleware tests (`scope_test.go`)
* [x] Fix expired key UTC bug in key service

## Sprint 3 — JWT Auth Foundation

* [ ] Migration 0004: users table
* [ ] User model
* [ ] User repository + Postgres implementation
* [ ] User service (register, login with bcrypt)
* [ ] JWT service (sign, validate, extract claims)
* [ ] Dependencies: `golang-jwt/jwt/v5`, `golang.org/x/crypto`
* [ ] Auth handler: POST /auth/login
* [ ] Auth middleware: validate JWT, attach to context
* [ ] Config: JWT_SECRET, ADMIN_EMAIL, ADMIN_PASSWORD
* [ ] First-run admin seeding in main.go
* [ ] Auth tests

## Sprint 4 — Permissions / RBAC

* [ ] Permission constants
* [ ] Migration 0005: roles + user_roles tables
* [ ] Role model + repository + Postgres implementation
* [ ] Role service (assign to user, get permissions)
* [ ] Permission middleware (RequirePermission)
* [ ] Seed admin role with all permissions
* [ ] Permission tests

## Sprint 5 — Scope Refactor + Route Protection

* [ ] Refactor BucketScope middleware (remove auth, keep scope)
* [ ] Apply Auth + Permission + Scope middleware to all routes
* [ ] Protect /buckets endpoints
* [ ] Protect /api-keys endpoints
* [ ] Updated scope tests

## Sprint 6 — Signed URLs

* [ ] Signed URL model
* [ ] Signed URL repository + migration
* [ ] Signed URL service
* [ ] Signed URL API
* [ ] Signed URL tests

## Sprint 7 — Standard API Responses

* [ ] Response models
* [ ] Response helpers
* [ ] Handler refactor
* [ ] Response tests

## Sprint 8 — Health Endpoints

* [ ] Liveness endpoint (JSON format)
* [ ] Readiness endpoint (DB + storage check)
* [ ] Health tests

## Sprint 9 — CLI Tool

* [ ] CLI project scaffold
* [ ] API client library
* [ ] Auth commands (login, token storage)
* [ ] Bucket commands
* [ ] Object commands
* [ ] CLI tests
