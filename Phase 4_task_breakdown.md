# Oort — Storage Enhancements

## Phase 4 Goal

At the end of Phase 4, this should work:

```bash
# Store objects on S3 instead of local disk
# (configured via environment variable)
export STORAGE_BACKEND=s3
export AWS_BUCKET=oort-prod

go run cmd/server/main.go
# Objects now live in S3, metadata in Postgres

# Upload and retrieve a specific version
curl -X PUT /buckets/docs/objects/report.pdf?v=1
curl -X PUT /buckets/docs/objects/report.pdf?v=2

# List all versions of an object
curl /buckets/docs/objects/report.pdf/versions

# Restore a previous version
curl -X POST /buckets/docs/objects/report.pdf/restore?v=1

# Upload a compressed object
curl -H "Content-Encoding: gzip" \
  -X PUT /buckets/docs/objects/data.json.gz \
  --data-binary @data.gz

# Compressed storage saves disk space
curl -H "Accept-Encoding: gzip" \
  /buckets/docs/objects/data.json.gz
# Server decompresses transparently

# Deduplicated storage — same content stored once
curl -X PUT /buckets/team/docs/contract.pdf -T contract.pdf
curl -X PUT /buckets/backup/contract.pdf -T contract.pdf
# Same SHA256 -> same blob on disk
```

And:

- Storage backend is pluggable (local, S3, GCS, Azure)
- Objects can be versioned per bucket
- Past versions can be listed and restored
- Compression happens transparently on upload/download
- Duplicate content is detected and stored once
- Storage savings are measurable

---

## Dependency Graph

```text
Phase 3
    |
    v
New StorageDriver Interface (extended)
    |
    +--------+--------+--------+
    |        |        |        |
    v        v        v        v
  Local    S3      GCS     Azure
  (exists) (new)   (new)   (new)
    |        |        |        |
    +--------+--------+--------+
              |
              v
   Storage Backend Factory
              |
    +---------+---------+
    |                   |
    v                   v
Version Model     ContentHash Model
    |                   |
    v                   v
Version Repo      Dedup Repository
    |                   |
    v                   v
Version Service   Dedup Service
    |                   |
    v                   v
Version API       Dedup Integration
    |                   |
    +---------+---------+
              |
    +---------+---------+
    |                   |
    v                   v
Compression       Dedup Backend
Middleware         (content-addressed)
    |
    v
Compression Config
```

---

# Epic 1 — Storage Abstraction Layer

Currently Oort is hardcoded to the local filesystem.

Without abstraction:

- Every backend requires changing `LocalDriver`
- Impossible to use cloud storage
- Testing requires real disk I/O
- Vendor lock-in to local disk

---

## Task 1.1 — Expanded StorageDriver Interface

The current interface is minimal. Expand it for production use.

**File:** `internal/storage/filesystem/driver.go`

```go
type StorageDriver interface {
    // Core operations
    Save(path string, r io.Reader) error
    Open(path string) (io.ReadCloser, error)
    Delete(path string) error

    // Introspection
    Exists(path string) (bool, error)
    Size(path string) (int64, error)

    // Management
    List(prefix string) ([]string, error)
    Copy(src, dst string) error

    // Lifecycle
    Close() error
}
```

---

## Task 1.2 — S3 Backend

**File:** `internal/storage/backends/s3.go`

```go
type S3Driver struct {
    client   *s3.Client
    bucket   string
    prefix   string
}
```

**Dependencies:**

```text
github.com/aws/aws-sdk-go-v2
github.com/aws/aws-sdk-go-v2/config
github.com/aws/aws-sdk-go-v2/service/s3
```

**Responsibilities:**

- Map local-style paths to S3 object keys under a prefix
- `Save` -> `PutObject` with streaming body
- `Open` -> `GetObject` returning body as ReadCloser
- `Delete` -> `DeleteObject`
- `Exists` -> `HeadObject`
- `Size` -> `HeadObject` -> ContentLength
- `List` -> `ListObjectsV2` with prefix
- `Copy` -> `CopyObject`

---

## Task 1.3 — GCS Backend

**File:** `internal/storage/backends/gcs.go`

```go
type GCSDriver struct {
    client    *storage.Client
    bucket    string
    prefix    string
}
```

**Dependencies:**

```text
cloud.google.com/go/storage
google.golang.org/api/option
```

**Responsibilities:**

- Same mapping as S3, but with GCS APIs
- `Save` -> `Writer` with streaming
- `Open` -> `NewReader`
- `Delete` -> `ObjectHandle.Delete`
- `Exists` -> `ObjectHandle.Attrs`
- `Size` -> `Attrs.Size`
- `List` -> `ObjectIterator`
- `Copy` -> `CopierFrom`

---

## Task 1.4 — Azure Blob Backend

**File:** `internal/storage/backends/azure.go`

```go
type AzureDriver struct {
    client    *blob.Client
    container string
    prefix    string
}
```

**Dependencies:**

```text
github.com/Azure/azure-sdk-for-go/sdk/storage/azblob
```

**Responsibilities:**

- Same mapping as S3, but with Azure Blob APIs
- `Save` -> `UploadStream`
- `Open` -> `DownloadStream`
- `Delete` -> `DeleteBlob`
- `Exists` -> `GetProperties`
- `Size` -> `Properties.ContentLength`
- `List` -> `ListBlobsFlat`
- `Copy` -> `StartCopyFromURL`

---

## Task 1.5 — Backend Factory

**File:** `internal/storage/backends/factory.go`

```go
func NewStorageDriver(cfg *config.Config) (StorageDriver, error) {
    switch cfg.StorageBackend {
    case "local":
        return filesystem.NewLocalDriver(cfg.BlobRoot)
    case "s3":
        return NewS3Driver(...)
    case "gcs":
        return NewGCSDriver(...)
    case "azure":
        return NewAzureDriver(...)
    default:
        return nil, fmt.Errorf("unknown backend: %s", cfg.StorageBackend)
    }
}
```

**Config additions:**

```env
STORAGE_BACKEND=local
BLOB_ROOT=./data/blobs

# S3
AWS_REGION=us-east-1
AWS_BUCKET=oort-storage
AWS_ACCESS_KEY_ID=xxx
AWS_SECRET_ACCESS_KEY=yyy

# GCS
GCS_BUCKET=oort-storage
GOOGLE_APPLICATION_CREDENTIALS=/path/to/key.json

# Azure
AZURE_STORAGE_ACCOUNT=oortstorage
AZURE_CONTAINER=oort-blobs
AZURE_ACCESS_KEY=xxx
```

---

## Task 1.6 — Backend Tests

**Verify (for each backend):**

- Save and read round-trip
- Delete removes blob
- Exists returns correct state
- Size returns correct bytes
- List with prefix works
- Copy duplicates blob
- Close cleans up resources

---

# Epic 2 — Object Versioning

Currently uploading to an existing key overwrites it silently.

Without versioning:

- Lost data is unrecoverable
- No audit trail of changes
- Cannot roll back to previous state

---

## Task 2.1 — Version Model

**File:** `internal/storage/models/version.go`

```go
type ObjectVersion struct {
    ID          uuid.UUID
    ObjectID    uuid.UUID
    VersionNum  int
    StoragePath string
    ContentType string
    SizeBytes   int64
    Checksum    string
    IsLatest    bool
    CreatedAt   time.Time
}
```

**Migration:**

```sql
CREATE TABLE object_versions (
    id UUID PRIMARY KEY,
    object_id UUID REFERENCES objects(id),
    version_num INT NOT NULL,
    storage_path TEXT NOT NULL,
    content_type TEXT,
    size_bytes BIGINT,
    checksum TEXT,
    is_latest BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(object_id, version_num)
);
```

Changes to existing `objects` table:

```sql
-- objects table now represents the "current" version
-- Add versioning_enabled flag to buckets
ALTER TABLE buckets ADD COLUMN versioning_enabled BOOLEAN DEFAULT FALSE;
```

---

## Task 2.2 — Version Repository

**File:** `internal/storage/repositories/version_repository.go`

**Methods:**

```go
CreateVersion(ctx, version *ObjectVersion) error
GetVersion(ctx, objectID uuid.UUID, versionNum int) (*ObjectVersion, error)
GetLatestVersion(ctx, objectID uuid.UUID) (*ObjectVersion, error)
ListVersions(ctx, objectID uuid.UUID) ([]ObjectVersion, error)
SetLatest(ctx, objectID uuid.UUID, versionNum int) error
DeleteVersion(ctx, versionID uuid.UUID) error
```

---

## Task 2.3 — Version Service

**Flow on upload (versioned bucket):**

```
PUT /objects/{key}
    |
    v
Create New Object Blob
    |
    v
Increment Version Number
    |
    v
Version 1: Copy blob to versioned path
    |
    v
Create ObjectVersion record (is_latest=true)
    |
    v
Unset previous latest flag
    |
    v
Update Object metadata to new blob
    |
    v
Old blob kept (previous version still exists)
```

**Flow on delete (versioned bucket):**

```
DELETE /objects/{key}
    |
    v
Mark Object as deleted (soft delete)
    |
    v
Add DeleteMarker version
    |
    v
Object no longer returned in GET/LIST
    |
    v
But all versions remain on disk
```

**Flow on restore:**

```
POST /objects/{key}/restore?v=1
    |
    v
Get Version 1
    |
    v
Copy version 1 blob to current path
    |
    v
Create New Version pointing to copied blob
    |
    v
Set as latest
```

**Business rules:**

- Versioning is per-bucket (opt-in)
- Non-versioned buckets behave like Phase 1 (overwrite + delete permanently)
- Listing returns only the latest version (unless `?versions` is specified)
- Deleting in a versioned bucket creates a delete marker (not permanent)
- Permanently delete only via `DELETE ?versionId={id}`

---

## Task 2.4 — Version API

**Endpoints:**

| Method | Path | Description |
|--------|------|-------------|
| `PUT` | `/buckets/{bucket}/versioning` | Enable/disable versioning |
| `GET` | `/buckets/{bucket}/objects/{key}/versions` | List all versions |
| `GET` | `/buckets/{bucket}/objects/{key}?version={n}` | Get specific version |
| `POST` | `/buckets/{bucket}/objects/{key}/restore?version={n}` | Restore version |
| `DELETE` | `/buckets/{bucket}/objects/{key}?versionId={id}` | Permanently delete version |

---

## Task 2.5 — Version Tests

**Verify:**

- Upload creates version history
- Listing returns all versions
- Get by version returns correct content
- Restore creates new version with old content
- Delete marker hides object
- Permanent delete removes specific version
- Non-versioned bucket unaffected

---

# Epic 3 — Compression

Large files consume disk space unnecessarily.

Without compression:

- Text files stored uncompressed
- Logs, JSON, CSVs waste space
- Bandwidth wasted on transfer
- No configuration for compression policy

---

## Task 3.1 — Compression Config

**File:** `internal/storage/models/compression.go`

```go
type CompressionPolicy struct {
    BucketID    uuid.UUID
    Algorithm   string // "gzip" | "zstd" | "none"
    MinSizeBytes int64  // only compress files above this size
}
```

**Migration:**

```sql
CREATE TABLE bucket_compression (
    bucket_id UUID PRIMARY KEY REFERENCES buckets(id),
    algorithm TEXT DEFAULT 'none',
    min_size_bytes BIGINT DEFAULT 1024
);

ALTER TABLE objects ADD COLUMN storage_encoding TEXT DEFAULT 'identity';
```

**Config:**

```env
COMPRESSION_ALGORITHM=gzip
COMPRESSION_MIN_SIZE=1024
```

---

## Task 3.2 — Compression Service

**File:** `internal/storage/services/compression_service.go`

```go
type CompressionService struct {
    algorithm string
    minSize   int64
}

func (s *CompressionService) Compress(r io.Reader) (io.ReadCloser, error)
func (s *CompressionService) Decompress(r io.Reader) (io.ReadCloser, error)
func (s *CompressionService) ContentEncoding() string
```

**Supported algorithms:**

- `gzip` — standard, good compatibility
- `zstd` — better ratio, faster, newer
- `none` — pass-through

**Integration with upload pipeline:**

```
PUT /upload
    |
    v
Read Content-Encoding header
    |
    +--> If "gzip", store compressed, set storage_encoding="gzip"
    |
    +--> If "identity", check bucket policy:
            |
            +--> If compression enabled AND size > min:
                    Compress, store, set storage_encoding="gzip"
            |
            +--> Else: store raw, storage_encoding="identity"
```

**Integration with download pipeline:**

```
GET /download
    |
    v
Check Accept-Encoding header
    |
    +--> If client accepts gzip AND object stored as gzip:
            Stream compressed bytes directly (no decompress)
            Set Content-Encoding: gzip
    |
    +--> If client doesn't accept gzip:
            Decompress on-the-fly, stream uncompressed
```

---

## Task 3.3 — Compression API

**Endpoints:**

| Method | Path | Description |
|--------|------|-------------|
| `PUT` | `/buckets/{bucket}/compression` | Set bucket compression policy |
| `GET` | `/buckets/{bucket}/compression` | Get bucket compression policy |
| `DELETE` | `/buckets/{bucket}/compression` | Disable compression |

---

## Task 3.4 — Compression Tests

**Verify:**

- Upload + download round-trip with compression
- Compressed object smaller than original
- Decompressed bytes match original
- Client without Accept-Encoding: gzip gets raw bytes
- Client with Accept-Encoding: gzip gets compressed bytes
- Compression policy respected per-bucket
- Objects under min size not compressed

---

# Epic 4 — Deduplication

Identical content stored multiple times wastes space.

Without dedup:

- Same file uploaded to 10 buckets = 10 copies
- CI artifacts duplicated across projects
- No content-aware storage

---

## Task 4.1 — Content Hash Model

**File:** `internal/storage/models/dedup.go`

```go
type ContentHash struct {
    SHA256      string `json:"sha256"`
    RefCount    int    `json:"ref_count"`
    StoragePath string `json:"storage_path"`
    SizeBytes   int64  `json:"size_bytes"`
    CreatedAt   time.Time `json:"created_at"`
}
```

**Migration:**

```sql
CREATE TABLE content_hashes (
    sha256 TEXT PRIMARY KEY,
    ref_count INT DEFAULT 1,
    storage_path TEXT NOT NULL,
    size_bytes BIGINT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Reference counting table
CREATE TABLE object_content_refs (
    object_id UUID REFERENCES objects(id),
    sha256 TEXT REFERENCES content_hashes(sha256),
    PRIMARY KEY (object_id, sha256)
);
```

---

## Task 4.2 — Dedup Repository

**File:** `internal/storage/repositories/dedup_repository.go`

**Methods:**

```go
FindByHash(ctx, sha256 string) (*ContentHash, error)
CreateHash(ctx, hash *ContentHash) error
IncrementRef(ctx, sha256 string) error
DecrementRef(ctx, sha256 string) (refCount int, err error)
DeleteHash(ctx, sha256 string) error
AddObjectRef(ctx, objectID uuid.UUID, sha256 string) error
RemoveObjectRef(ctx, objectID uuid.UUID, sha256 string) error
```

---

## Task 4.3 — Dedup Service

**Integration with upload pipeline:**

```
PUT /upload
    |
    v
Stream body -> Calculate SHA256 (tee)
    |
    v
Check content_hashes table
    |
    +--> Hash exists:
    |       |
    |       +--> Increment ref_count
    |       |
    |       +--> Add object_content_refs row
    |       |
    |       +--> Discard blob (already stored)
    |       |
    |       +--> Return success
    |
    +--> Hash does not exist:
            |
            +--> Write blob to storage
            |
            +--> Create content_hashes row (ref_count=1)
            |
            +--> Add object_content_refs row
            |
            +--> Return success
```

**Integration with delete pipeline:**

```
DELETE /object
    |
    v
Lookup content hash
    |
    v
Decrement ref_count
    |
    +--> ref_count == 0:
    |       |
    |       +--> Delete blob from storage
    |       |
    |       +--> Delete content_hashes row
    |
    +--> ref_count > 0:
            |
            +--> Do nothing (other objects still reference it)
    |
    v
Delete object_content_refs row
    |
    v
Delete object metadata
```

**Content-addressed storage paths:**

```text
data/blobs/ca/978f2a...   (stored by SHA256, not UUID)
data/blobs/3b/5d5eac...   (first two chars as shard)
```

---

## Task 4.4 — Dedup API

**Endpoints:**

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/system/stats/dedup` | Get dedup statistics |

**Response:**

```json
{
  "total_objects": 1500,
  "unique_content": 1200,
  "saved_bytes": 524288000,
  "savings_pct": 20.0
}
```

---

## Task 4.5 — Dedup Garbage Collection

**Background worker to reconcile ref counts:**

```go
type DedupGC struct {
    repo    DedupRepository
    storage StorageDriver
}

func (g *DedupGC) Run(ctx context.Context) {
    // 1. Find all content_hashes
    // 2. For each, verify ref_count matches actual refs in object_content_refs
    // 3. If ref_count is stale, correct it
    // 4. If ref_count == 0, delete blob and row
}
```

**WAL (Write-Ahead Logging) for crash safety:**

```text
Before deleting any blob, log intent:
  intent/delete/2026-01-15/ca978f2a...
  
After blob deleted, remove intent file.
On restart, process any lingering intent files.
```

---

## Task 4.6 — Dedup Tests

**Verify:**

- Same content uploaded twice -> single blob
- Ref count increments correctly
- Delete decrements and cleans up at zero
- Garbage collection reconciles ref counts
- Dedup stats accurate
- Concurrent upload of same content handled safely

---

# Actual Execution Order

Don't jump around. Do them in this exact order:

## Sprint 1

- [ ] Expand StorageDriver interface
- [ ] Implement S3 backend
- [ ] Implement GCS backend
- [ ] Implement Azure backend
- [ ] Backend factory
- [ ] Backend tests for each driver

## Sprint 2

- [ ] Version model + migration
- [ ] Version repository
- [ ] Version service
- [ ] Version API (enable, list, get by version)
- [ ] Restore + soft delete
- [ ] Version tests

## Sprint 3

- [ ] Compression config + model
- [ ] Compression service (gzip, zstd)
- [ ] Upload pipeline integration
- [ ] Download pipeline integration (transparent)
- [ ] Compression API
- [ ] Compression tests

## Sprint 4

- [ ] Content hash model + migration
- [ ] Dedup repository
- [ ] Dedup service (upload + delete integration)
- [ ] Content-addressed storage paths
- [ ] Dedup GC worker
- [ ] Dedup tests
