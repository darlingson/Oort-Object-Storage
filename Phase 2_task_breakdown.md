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
* No ownership model

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
    CreatedAt   time.Time
    LastUsedAt  *time.Time
}
```

---

## Task 1.2 — API Key Repository

Responsibilities:

* Create key
* Find by hash
* List keys
* Delete key

Methods:

```go
Create(...)
FindByHash(...)
List(...)
Delete(...)
```

---

## Task 1.3 — API Key Service

Business Rules:

* Generate secure keys
* Never store raw key
* Store SHA256 hash only
* Return raw key once

Example:

```text
oort_f3b2f8...
```

---

## Task 1.4 — API Key API

Endpoints:

| Method | Endpoint       |
| ------ | -------------- |
| POST   | /api-keys      |
| GET    | /api-keys      |
| DELETE | /api-keys/{id} |

---

## Task 1.5 — API Key Tests

Verify:

* Key creation
* Hash stored
* Raw key not stored
* Delete works

---

# Epic 2 — Authentication Middleware

Protect write operations.

---

## Task 2.1 — Authentication Middleware

Responsibilities:

* Read Authorization header
* Validate API key
* Attach identity to request context

Example:

```http
Authorization: Bearer oort_xxx
```

---

## Task 2.2 — Route Protection

Require authentication for:

```text
PUT object
DELETE object
POST bucket
DELETE bucket (future)
```

Allow:

```text
GET health
GET signed url downloads
```

---

## Task 2.3 — Middleware Tests

Verify:

* Missing key rejected
* Invalid key rejected
* Valid key accepted

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

* [ ] API key model
* [ ] API key repository
* [ ] API key service
* [ ] API key API
* [ ] API key tests

## Sprint 2

* [ ] Authentication middleware
* [ ] Route protection
* [ ] Middleware tests

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