# Oort — Object Storage Service

## Phase 1 Goal

At the end of Phase 1, this should work:

```bash
# Create a bucket
curl -X POST /buckets/documents

# Upload a file
curl -T contract.pdf \
  /buckets/documents/objects/contracts/contract.pdf

# Download a file
curl \
  /buckets/documents/objects/contracts/contract.pdf
```

And:

- File exists on disk
- Metadata exists in Postgres
- Server can restart
- File still downloads

---

## Dependency Graph

```
Database
  |
  v
Bucket Model
  |
  v
Bucket Service
  |
  v
Bucket API
  |
  +----------------------+
  |                      |
  v                      |
Filesystem Driver        |
  |                      |
  v                      |
Object Model             |
  |                      |
  v                      |
Object Service           |
  |                      |
  v                      |
Upload API               |
  |                      |
  v                      |
Download API             |
  |                      |
  v                      |
Delete API <-------------+
```

---

## Epic 1 — Bucket Management

Everything depends on buckets.

Without buckets:

- Uploads don't know where to go
- Objects have nowhere to belong

### Task 1.1 — Bucket Model

**File:** `internal/storage/models/bucket.go`

```go
type Bucket struct {
    ID        string
    Name      string
    CreatedAt time.Time
}
```

### Task 1.2 — Bucket Repository

**Responsibilities:**

- Create bucket
- Find bucket
- List buckets

**File:** `internal/storage/repositories/bucket_repository.go`

**Methods:**

- `Create(...)`
- `FindByName(...)`
- `List(...)`

### Task 1.3 — Bucket Service

**Business rules:**

- Bucket names unique
- Bucket names valid
- Cannot create duplicates

**File:** `internal/services/bucket_service.go`

### Task 1.4 — Bucket API

**Endpoints:**

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/buckets` | Create a new bucket |
| `GET`  | `/buckets` | List all buckets |
| `GET`  | `/buckets/{name}` | Get a specific bucket |

### Task 1.5 — Bucket Tests

**Test:**

- Create bucket
- Duplicate bucket rejected
- List buckets

---

## Epic 2 — Filesystem Driver

After buckets exist.

### Task 2.1 — Storage Driver Interface

You already started this.

```go
type StorageDriver interface {
    Save(...)
    Open(...)
    Delete(...)
}
```

### Task 2.2 — Local Filesystem Implementation

**File:** `internal/storage/filesystem/local.go`

**Responsibilities:**

- Create directories
- Write blobs
- Read blobs
- Delete blobs

### Task 2.3 — Storage Tests

**Verify:**

- Save file
- Read file
- Delete file

No database involved.

---

## Epic 3 — Object Metadata

Now we can track files.

### Task 3.1 — Object Model

```go
type Object struct {
    ID          string
    BucketID    string
    ObjectKey   string
    StoragePath string
    ContentType string
    SizeBytes   int64
    Checksum    string
    CreatedAt   time.Time
}
```

### Task 3.2 — Object Repository

**Methods:**

- `Create(...)`
- `Find(...)`
- `Delete(...)`
- `List(...)`

### Task 3.3 — Object Repository Tests

**Verify:** metadata persistence.

---

## Epic 4 — Upload Pipeline

This is where Oort becomes real.

### Task 4.1 — Upload Service

**Flow:**

```
HTTP Request
    |
    v
Validate Bucket
    |
    v
Generate UUID
    |
    v
Write Blob To Disk
    |
    v
Create Metadata Record
    |
    v
Return Success
```

### Task 4.2 — Upload Endpoint

| Method | Path | Description |
|--------|------|-------------|
| `PUT` | `/buckets/{bucket}/objects/{key}` | Upload an object |

**Example:**

```
PUT /buckets/documents/objects/contracts/file.pdf
```

### Task 4.3 — Upload Tests

**Verify:**

- Metadata created
- File exists on disk

---

## Epic 5 — Download Pipeline

### Task 5.1 — Download Service

**Flow:**

```
Lookup Metadata
    |
    v
Open File
    |
    v
Stream Response
```

### Task 5.2 — Download Endpoint

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/buckets/{bucket}/objects/{key}` | Download an object |

### Task 5.3 — Download Tests

**Verify:**

- Bytes match upload

---

## Epic 6 — Delete Pipeline

### Task 6.1 — Delete Service

**Flow:**

```
Find Metadata
    |
    v
Delete File
    |
    v
Delete Metadata
```

### Task 6.2 — Delete Endpoint

| Method | Path | Description |
|--------|------|-------------|
| `DELETE` | `/buckets/{bucket}/objects/{key}` | Delete an object |

### Task 6.3 — Delete Tests

**Verify:**

- File removed
- Metadata removed

---

## Epic 7 — Integrity

Small but important.

### Task 7.1 — SHA256 Generation

During upload:

```
Stream
  |
  +--> Disk
  |
  +--> SHA256
```

Store hash in DB.

### Task 7.2 — Integrity Tests

**Verify:** checksum stored.

---

## Actual Execution Order

Don't jump around. Do them in this exact order:

### Sprint 1
- [ ] Bucket model
- [ ] Bucket repository
- [ ] Bucket service
- [ ] Bucket API
- [ ] Bucket tests

### Sprint 2
- [ ] Storage driver interface
- [ ] Local filesystem driver
- [ ] Storage tests

### Sprint 3
- [ ] Object model
- [ ] Object repository
- [ ] Repository tests

### Sprint 4
- [ ] Upload service
- [ ] Upload API
- [ ] Upload tests

### Sprint 5
- [ ] Download service
- [ ] Download API
- [ ] Download tests

### Sprint 6
- [ ] Delete service
- [ ] Delete API
- [ ] Delete tests

### Sprint 7
- [ ] SHA256 integrity
- [ ] Integrity tests