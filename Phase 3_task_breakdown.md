# Oort — Production Features

## Phase 3 Goal

At the end of Phase 3, this should work:

```bash
# Upload a large file in parts
curl -X POST /buckets/videos/objects/movie.mp4/uploads

# Upload part 1 of 4
curl -X PUT \
  /buckets/videos/objects/movie.mp4/uploads/{uploadId}/parts/1 \
  -T movie.part1

# Complete the multipart upload
curl -X POST \
  /buckets/videos/objects/movie.mp4/uploads/{uploadId}/complete

# Resume an interrupted upload
curl -X GET \
  /buckets/videos/objects/movie.mp4/uploads/{uploadId}/status
# ...continues from last successful byte

# Verify checksum
curl -X POST \
  /buckets/videos/objects/movie.mp4/verify

# Set retention policy (auto-delete after 30 days)
curl -X PUT /buckets/videos/retention \
  -H "Content-Type: application/json" \
  -d '{"policy": "retain_for_days", "value": 30}'

# Set bucket CORS policy
curl -X PUT /buckets/videos/cors \
  -H "Content-Type: application/json" \
  -d '{"allowed_origins": ["https://app.example.com"]}'
```

And:

- Large files (5GB+) can be uploaded in parallel parts
- Interrupted uploads can resume without re-sending bytes
- Checksums are verified on download; corruption is detectable
- Objects expire automatically based on retention rules
- Buckets have configurable CORS, IP, and referer policies
- Nothing breaks for objects that don't use these features

---

## Dependency Graph

```text
Phase 1 & 2
    |
    v
Multipart Upload Model
    |
    v
Upload Session Repository
    |
    v
Multipart Upload Service <----+
    |                          |
    v                          |
Multipart Upload API           |
    |                          |
    +--------------------------+
    |
    v
Resumable Upload Session Model
    |
    v
Resumable Upload Repository
    |
    v
Resumable Upload Service
    |
    v
Resumable Upload API
    |
    +--------------------------+
    |                          |
    v                          |
Checksum Service               |
    |                          |
    v                          |
Verify API                     |
    |                          |
    +--------------------------+
    |
    v
Retention Policy Model
    |
    v
Retention Policy Repository
    |
    v
Retention Service
    |
    v
Retention API
    |
    +--------------------------+
    |
    v
Bucket Policy Models (CORS, IP, Referer)
    |
    v
Bucket Policy Repository
    |
    v
Bucket Policy Service
    |
    v
Bucket Policy API
```

---

# Epic 1 — Multipart Uploads

Single-shot uploads fail for large files.

Without multipart:

- 5GB upload ties up connection for hours
- One network blip means starting over
- No progress visibility

---

## Task 1.1 — Multipart Upload Model

**File:** `internal/storage/models/multipart_upload.go`

```go
type MultipartUpload struct {
    ID           uuid.UUID
    BucketID     uuid.UUID
    ObjectKey    string
    ContentType  string
    InitiatedAt  time.Time
    CompletedAt  *time.Time
    Status       string // "initiated" | "completed" | "aborted"
}

type UploadPart struct {
    ID           uuid.UUID
    UploadID     uuid.UUID
    PartNumber   int
    StoragePath  string
    SizeBytes    int64
    Checksum     string
    CreatedAt    time.Time
}
```

**Migration:**

```sql
CREATE TABLE multipart_uploads (
    id UUID PRIMARY KEY,
    bucket_id UUID REFERENCES buckets(id),
    object_key TEXT NOT NULL,
    content_type TEXT,
    initiated_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP,
    status TEXT DEFAULT 'initiated'
);

CREATE TABLE upload_parts (
    id UUID PRIMARY KEY,
    upload_id UUID REFERENCES multipart_uploads(id),
    part_number INT NOT NULL,
    storage_path TEXT NOT NULL,
    size_bytes BIGINT,
    checksum TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(upload_id, part_number)
);
```

---

## Task 1.2 — Multipart Upload Repository

**File:** `internal/storage/repositories/multipart_upload_repository.go`

**Methods:**

```go
CreateUpload(ctx, upload *MultipartUpload) error
FindUpload(ctx, uploadID uuid.UUID) (*MultipartUpload, error)
CompleteUpload(ctx, uploadID uuid.UUID) error
AbortUpload(ctx, uploadID uuid.UUID) error

SavePart(ctx, part *UploadPart) error
ListParts(ctx, uploadID uuid.UUID) ([]UploadPart, error)
DeleteParts(ctx, uploadID uuid.UUID) error
```

---

## Task 1.3 — Multipart Upload Service

**Flow:**

```
POST /initiate
    |
    v
Create Upload Session (DB)
    |
    v
Return Upload ID
    |
    ...

PUT /upload/{uploadId}/parts/{partNumber}
    |
    v
Validate Session Status = "initiated"
    |
    v
Save Part Blob To Disk
    |
    v
Save Part Metadata (DB)
    |
    v
Return ETag (part checksum)
    |
    ...

POST /complete
    |
    v
Validate All Parts Present
    |
    v
Assemble Final Object (concatenate parts)
    |
    v
Write Complete Object Blob
    |
    v
Delete Individual Part Blobs
    |
    v
Create Object Metadata Record
    |
    v
Mark Upload Completed
```

**Business rules:**

- Max part count: 10,000 (S3-compatible)
- Min part size: 5MB (except last part)
- Parts must be contiguous (no gaps)
- Cannot complete an already-completed upload
- Abort deletes all part blobs and metadata

---

## Task 1.4 — Multipart Upload API

**Endpoints:**

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/buckets/{bucket}/objects/{key}/uploads` | Initiate multipart upload |
| `PUT` | `/buckets/{bucket}/objects/{key}/uploads/{uploadId}/parts/{partNumber}` | Upload a part |
| `POST` | `/buckets/{bucket}/objects/{key}/uploads/{uploadId}/complete` | Complete multipart upload |
| `DELETE` | `/buckets/{bucket}/objects/{key}/uploads/{uploadId}` | Abort multipart upload |
| `GET` | `/buckets/{bucket}/objects/{key}/uploads/{uploadId}/parts` | List uploaded parts |

---

## Task 1.5 — Multipart Upload Tests

**Verify:**

- Initiate creates session
- Part upload persists
- Complete assembles object correctly
- Abort cleans up blobs and metadata
- Missing parts rejected on complete
- Duplicate part numbers overwrite (last wins)
- Concurrent part uploads work

---

# Epic 2 — Resumable Uploads

Even with multipart, a single part can fail mid-transfer.

Without resumable uploads:

- Dropped connection loses partial part bytes
- Client must re-upload entire part
- Unreliable on mobile / spotty networks

---

## Task 2.1 — Resumable Upload Model

**File:** `internal/storage/models/resumable_upload.go`

```go
type ResumableUpload struct {
    ID              uuid.UUID
    BucketID        uuid.UUID
    ObjectKey       string
    ContentType     string
    TotalBytes      int64
    UploadedBytes   int64
    StoragePath     string
    Status          string // "in_progress" | "completed" | "expired"
    ExpiresAt       time.Time
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

**Migration:**

```sql
CREATE TABLE resumable_uploads (
    id UUID PRIMARY KEY,
    bucket_id UUID REFERENCES buckets(id),
    object_key TEXT NOT NULL,
    content_type TEXT,
    total_bytes BIGINT NOT NULL,
    uploaded_bytes BIGINT DEFAULT 0,
    storage_path TEXT NOT NULL,
    status TEXT DEFAULT 'in_progress',
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

---

## Task 2.2 — Resumable Upload Repository

**File:** `internal/storage/repositories/resumable_upload_repository.go`

**Methods:**

```go
Create(ctx, upload *ResumableUpload) error
FindByID(ctx, id uuid.UUID) (*ResumableUpload, error)
UpdateProgress(ctx, id uuid.UUID, uploadedBytes int64) error
Complete(ctx, id uuid.UUID) error
FindExpired(ctx) ([]ResumableUpload, error)
Delete(ctx, id uuid.UUID) error
```

---

## Task 2.3 — Resumable Upload Service

**Flow:**

```
PUT /resume
    |
    v
Create Temp File (storage_path)
    |
    v
Create Upload Session (DB)
    |
    v
Return Upload ID
    |
    ...

PATCH /resume/{uploadId}
    |
    v
Read offset from Content-Range header
    |
    v
Validate offset == uploaded_bytes
    |
    v
Append bytes to temp file
    |
    v
Update uploaded_bytes in DB
    |
    v
Return new offset (200) or complete (201)
    |
    ...

GET /resume/{uploadId}/status
    |
    v
Return uploaded_bytes, total_bytes, status
```

**Business rules:**

- Upload session expires after 24 hours (configurable)
- Offset must match server-side `uploaded_bytes`
- Overlapping ranges rejected
- Final byte count must match `total_bytes`
- On completion, move temp file to final blob path and create metadata record
- Periodic cleanup of expired sessions

---

## Task 2.4 — Resumable Upload API

**Endpoints:**

| Method | Path | Description |
|--------|------|-------------|
| `PUT` | `/buckets/{bucket}/objects/{key}/resume` | Initiate resumable upload |
| `PATCH` | `/buckets/{bucket}/objects/{key}/resume/{uploadId}` | Upload a chunk |
| `GET` | `/buckets/{bucket}/objects/{key}/resume/{uploadId}` | Get upload status |

**Headers:**

```http
Content-Range: bytes 0-1048575/10485760
Content-Length: 1048576
```

**Response:**

```http
200 Range: bytes=0-1048575
201 (completed, object metadata in body)
```

---

## Task 2.5 — Resumable Upload Tests

**Verify:**

- Initiation creates session
- Chunk append succeeds
- Offset mismatch rejected
- Completion creates object
- Expired sessions cleaned up
- Overlapping chunk rejected

---

# Epic 3 — Checksum Verification

Right now SHA256 is stored but never checked on download.

Without verification:

- Silent corruption goes undetected
- Disk bit-rot propagates to clients
- No trust in data integrity

---

## Task 3.1 — Verify-on-Download Service

**Flow:**

```
GET /download
    |
    v
Lookup Object Metadata
    |
    v
Open File Stream
    |
    v
Calculate SHA256 While Streaming
    |
    v
Compare to Stored Checksum
    |
    +--> Match: Stream to Client (200)
    +--> Mismatch: Return 500 (corruption detected)
```

**File:** `internal/services/checksum_service.go` (add to existing)

**Signature:**

```go
func (s *ObjectService) DownloadWithVerification(
    ctx context.Context, bucketName, objectKey string,
) (io.ReadCloser, *models.Object, error)
```

The reader must tee into a hasher before writing to the HTTP response. The checksum comparison happens after the full stream is sent.

---

## Task 3.2 — Periodic Integrity Check

**Background worker that scans all objects and verifies checksums.**

**File:** `internal/services/integrity_checker.go`

```go
type IntegrityChecker struct {
    objects    ObjectRepository
    storage    StorageDriver
    interval   time.Duration
}

func (c *IntegrityChecker) Run(ctx context.Context) {
    ticker := time.NewTicker(c.interval)
    for range ticker.C {
        c.scanAllObjects(ctx)
    }
}

func (c *IntegrityChecker) scanAllObjects(ctx context.Context) {
    // 1. List all buckets
    // 2. For each bucket, list all objects
    // 3. Open storage file, compute SHA256
    // 4. Compare to stored checksum
    // 5. Log mismatches as corruption events
}
```

**Output:**

```text
Corruption detected: bucket=documents key=contract.pdf
  expected: a1b2c3...  actual: d4e5f6...
```

---

## Task 3.3 — Checksum Verification API

**Endpoints:**

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/buckets/{bucket}/objects/{key}/checksum` | Get stored checksum |
| `POST` | `/buckets/{bucket}/objects/{key}/verify` | Trigger integrity check |

**Verify response:**

```json
{
  "checksum": "a1b2c3...",
  "verified": true,
  "checked_at": "2026-01-15T10:30:00Z"
}
```

---

## Task 3.4 — Integrity Tests

**Verify:**

- Corrupted file detected on download
- Checksum endpoint returns stored hash
- Verify endpoint detects corruption
- Integrity checker scans all objects
- Missing blob reported

---

# Epic 4 — Retention Policies

Objects accumulate forever.

Without retention:

- Old files never cleaned up
- Storage fills up
- Compliance requirements unmet

---

## Task 4.1 — Retention Model

**File:** `internal/storage/models/retention.go`

```go
type RetentionPolicy struct {
    BucketID  uuid.UUID
    Policy    string // "retain_for_days" | "retain_until" | "none"
    Value     int    // days or 0
}

type ObjectRetention struct {
    ObjectID        uuid.UUID
    Policy          string
    RetainUntil     time.Time
    LegalHold       bool
    CreatedAt       time.Time
}
```

**Migration:**

```sql
CREATE TABLE retention_policies (
    bucket_id UUID PRIMARY KEY REFERENCES buckets(id),
    policy TEXT NOT NULL DEFAULT 'none',
    value INT DEFAULT 0
);

CREATE TABLE object_retention (
    object_id UUID PRIMARY KEY REFERENCES objects(id),
    policy TEXT NOT NULL,
    retain_until TIMESTAMP NOT NULL,
    legal_hold BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW()
);
```

---

## Task 4.2 — Retention Repository

**File:** `internal/storage/repositories/retention_repository.go`

**Methods:**

```go
SetBucketPolicy(ctx, bucketID uuid.UUID, policy string, value int) error
GetBucketPolicy(ctx, bucketID uuid.UUID) (*RetentionPolicy, error)
DeleteBucketPolicy(ctx, bucketID uuid.UUID) error

SetObjectRetention(ctx, retention *ObjectRetention) error
GetObjectRetention(ctx, objectID uuid.UUID) (*ObjectRetention, error)
ListExpiredObjects(ctx) ([]uuid.UUID, error)
```

---

## Task 4.3 — Retention Service

**Flow (object-level):**

```
PUT /objects/{key}/retention
    |
    v
Validate Retention Period
    |
    v
Store ObjectRetention
    |
    v
Block Delete Until Expiry
```

**Flow (bucket-level lifecycle):**

```
Background Worker (every hour)
    |
    v
Find ObjectRetention WHERE retain_until < NOW()
    |
    v
For Each Expired Object:
    |
    +--> Delete Blob
    |
    +--> Delete Metadata
    |
    +--> Log Cleanup Event
```

**Business rules:**

- Objects under legal hold cannot be deleted regardless of expiry
- Bucket policy applies to all new objects by default
- Object-level policy overrides bucket policy
- Delete endpoint checks `retain_until` before proceeding
- Lifecycle worker logs all actions

---

## Task 4.4 — Retention API

**Endpoints:**

| Method | Path | Description |
|--------|------|-------------|
| `PUT` | `/buckets/{bucket}/retention` | Set bucket retention policy |
| `GET` | `/buckets/{bucket}/retention` | Get bucket retention policy |
| `DELETE` | `/buckets/{bucket}/retention` | Remove bucket retention policy |
| `PUT` | `/buckets/{bucket}/objects/{key}/retention` | Set object retention |
| `GET` | `/buckets/{bucket}/objects/{key}/retention` | Get object retention |

---

## Task 4.5 — Retention Tests

**Verify:**

- Bucket policy set and retrieved
- Object under retention cannot be deleted
- Expired objects cleaned up by worker
- Legal hold prevents deletion
- Object policy overrides bucket policy
- Lifecycle worker runnable standalone

---

# Epic 5 — Bucket Policies

Exposing object storage directly to the web without controls is risky.

Without bucket policies:

- Any origin can embed your files
- Any IP can reach your storage
- No referer restrictions
- CORS errors for legitimate web apps

---

## Task 5.1 — Bucket Policy Models

**File:** `internal/storage/models/bucket_policy.go`

```go
type CORSPolicy struct {
    BucketID        uuid.UUID
    AllowedOrigins  []string
    AllowedMethods  []string
    AllowedHeaders  []string
    ExposeHeaders   []string
    MaxAgeSeconds   int
}

type IPPolicy struct {
    BucketID    uuid.UUID
    AllowList   []string // CIDR notation
    DenyList    []string
}

type RefererPolicy struct {
    BucketID    uuid.UUID
    AllowEmpty  bool
    Whitelist   []string
}
```

**Migration:**

```sql
CREATE TABLE bucket_cors (
    bucket_id UUID PRIMARY KEY REFERENCES buckets(id),
    allowed_origins TEXT[] NOT NULL DEFAULT '{}',
    allowed_methods TEXT[] NOT NULL DEFAULT '{}',
    allowed_headers TEXT[] NOT NULL DEFAULT '{}',
    expose_headers TEXT[] NOT NULL DEFAULT '{}',
    max_age_seconds INT DEFAULT 3600
);

CREATE TABLE bucket_ip_policy (
    bucket_id UUID PRIMARY KEY REFERENCES buckets(id),
    allow_list TEXT[] NOT NULL DEFAULT '{}',
    deny_list TEXT[] NOT NULL DEFAULT '{}'
);

CREATE TABLE bucket_referer_policy (
    bucket_id UUID PRIMARY KEY REFERENCES buckets(id),
    allow_empty BOOLEAN DEFAULT TRUE,
    whitelist TEXT[] NOT NULL DEFAULT '{}'
);
```

---

## Task 5.2 — Bucket Policy Repository

**File:** `internal/storage/repositories/bucket_policy_repository.go`

**Methods:**

```go
// CORS
SetCORS(ctx, policy *CORSPolicy) error
GetCORS(ctx, bucketID uuid.UUID) (*CORSPolicy, error)
DeleteCORS(ctx, bucketID uuid.UUID) error

// IP
SetIPPolicy(ctx, policy *IPPolicy) error
GetIPPolicy(ctx, bucketID uuid.UUID) (*IPPolicy, error)
DeleteIPPolicy(ctx, bucketID uuid.UUID) error

// Referer
SetRefererPolicy(ctx, policy *RefererPolicy) error
GetRefererPolicy(ctx, bucketID uuid.UUID) (*RefererPolicy, error)
DeleteRefererPolicy(ctx, bucketID uuid.UUID) error
```

---

## Task 5.3 — Bucket Policy Middleware

**File:** `internal/api/middleware/bucket_policy.go`

The middleware runs on every bucket-scoped request:

```
Request -> Bucket Policy Middleware
    |
    +--> Check CORS (Origin header)
    |       |
    |       +--> Reject if origin not allowed
    |
    +--> Check IP Policy
    |       |
    |       +--> Reject if IP denied / not allowed
    |
    +--> Check Referer
    |       |
    |       +--> Reject if referer not allowed
    |
    v
Next Handler
```

**CORS response headers:**

```http
Access-Control-Allow-Origin: https://app.example.com
Access-Control-Allow-Methods: GET, PUT, DELETE
Access-Control-Allow-Headers: Content-Type, Authorization
Access-Control-Max-Age: 3600
```

---

## Task 5.4 — Bucket Policy API

**Endpoints:**

| Method | Path | Description |
|--------|------|-------------|
| `PUT` | `/buckets/{bucket}/cors` | Set CORS policy |
| `GET` | `/buckets/{bucket}/cors` | Get CORS policy |
| `DELETE` | `/buckets/{bucket}/cors` | Delete CORS policy |
| `PUT` | `/buckets/{bucket}/ip-policy` | Set IP policy |
| `GET` | `/buckets/{bucket}/ip-policy` | Get IP policy |
| `DELETE` | `/buckets/{bucket}/ip-policy` | Delete IP policy |
| `PUT` | `/buckets/{bucket}/referer` | Set referer policy |
| `GET` | `/buckets/{bucket}/referer` | Get referer policy |
| `DELETE` | `/buckets/{bucket}/referer` | Delete referer policy |

---

## Task 5.5 — Bucket Policy Tests

**Verify:**

- CORS headers returned on OPTIONS preflight
- Disallowed origin rejected
- IP allow list works
- IP deny list blocks specific addresses
- Referer whitelist works
- Empty referer handling
- Policy deletion clears restrictions

---

# Epic 6 — Object Lock (Legal Hold)

Compliance teams need immutable objects.

Without legal hold:

- Retention is time-based and eventually expires
- No way to indefinitely preserve evidence
- No audit trail for holds

---

## Task 6.1 — Object Lock Integration

Extends the retention system:

- Legal hold is independent of retention expiry
- Object under legal hold cannot be deleted **ever**
- Legal hold can only be added/removed by authorized API keys
- Audit log tracks hold placement and removal

---

## Task 6.2 — Object Lock API

**Endpoints:**

| Method | Path | Description |
|--------|------|-------------|
| `PUT` | `/buckets/{bucket}/objects/{key}/legal-hold` | Place legal hold |
| `DELETE` | `/buckets/{bucket}/objects/{key}/legal-hold` | Remove legal hold |
| `GET` | `/buckets/{bucket}/objects/{key}/legal-hold` | Check legal hold status |

---

## Task 6.3 — Object Lock Tests

**Verify:**

- Object under hold cannot be deleted
- Hold can be removed
- Hold removal authorized
- Audit event logged

---

# Actual Execution Order

Don't jump around. Do them in this exact order:

## Sprint 1

- [ ] Multipart upload model + migration
- [ ] Multipart upload repository
- [ ] Multipart upload service (init, part, complete, abort)
- [ ] Multipart upload API
- [ ] Multipart upload tests

## Sprint 2

- [ ] Resumable upload model + migration
- [ ] Resumable upload repository
- [ ] Resumable upload service
- [ ] Resumable upload API
- [ ] Resumable upload tests

## Sprint 3

- [ ] Verify-on-download service
- [ ] Periodic integrity checker
- [ ] Checksum verification API
- [ ] Integrity tests

## Sprint 4

- [ ] Retention models + migration
- [ ] Retention repository
- [ ] Retention service (object + lifecycle worker)
- [ ] Retention API
- [ ] Retention tests

## Sprint 5

- [ ] Bucket policy models + migrations
- [ ] Bucket policy repository
- [ ] Bucket policy middleware
- [ ] Bucket policy API
- [ ] Bucket policy tests

## Sprint 6

- [ ] Legal hold integration
- [ ] Legal hold API
- [ ] Legal hold tests
