# Oort Object Storage

Oort is a lightweight, self-hosted object storage server inspired by Amazon S3 and MinIO.

The goal is to provide a simple and reliable way to store, retrieve, and manage files through an HTTP API without relying on public cloud storage providers.

Oort is being built as a learning project focused on storage systems, streaming uploads, metadata management, and infrastructure engineering.

---

## Why Oort?

Many applications need a place to store files:

* Document management systems
* E-signature platforms
* Survey attachments
* Media libraries
* Internal enterprise applications
* AI document repositories

While cloud services like S3 work well, there are cases where organizations want:

* Self-hosted infrastructure
* Full control over data
* Local network deployments
* Lower operating costs
* Simpler deployments

Oort aims to provide those capabilities through a lightweight storage server that can run anywhere.

---

## Core Concepts

### Buckets

Buckets are logical containers for objects.

Examples:

```text
documents
backups
avatars
```

### Objects

Objects are files stored inside buckets.

Examples:

```text
documents/contracts/contract.pdf
avatars/users/john.jpg
```

### Metadata

Object metadata is stored in PostgreSQL and includes:

* Object key
* Bucket
* Content type
* File size
* Checksum
* Storage location
* Creation timestamps

### Blob Storage

Actual file contents are stored on disk.

Oort separates metadata from file storage:

```text
PostgreSQL -> Metadata
Filesystem -> File Contents
```

---

## Current Status

Project status: Early Development

### Implemented

* HTTP server
* PostgreSQL integration
* Configuration management
* Database migrations
* Automated tests

### In Progress

* Bucket management
* Filesystem storage driver
* Object upload pipeline

---

## Roadmap

### Phase 1: Core Storage Engine

* Create buckets
* List buckets
* Upload objects
* Download objects
* Delete objects
* Store metadata in PostgreSQL
* Local filesystem storage

### Phase 2: Developer Experience

* API keys
* Signed URLs
* Structured API responses
* CLI tool
* Health endpoints

### Phase 3: Production Features

* Multipart uploads
* Resumable uploads
* Checksum verification
* Retention policies
* Bucket policies

### Phase 4: Storage Enhancements

* Storage abstraction layer
* Object versioning
* Compression
* Deduplication

### Phase 5: Distributed Concepts

* Replication
* Multi-node deployments
* Cluster coordination

### Phase 6: Ecosystem

* Web UI
* SDKs
* Partial S3 compatibility
* Metrics and observability

---

## Architecture

```text
Client
   |
HTTP API
   |
Service Layer
   |
+--------------------+
|                    |
v                    v

PostgreSQL      Filesystem
(Metadata)      (Blobs)
```

---

## Technology Stack

* Go
* PostgreSQL
* Local Filesystem Storage
* Docker
* Standard Library HTTP Server

---

## Running Locally

### Requirements

* Go 1.25+
* PostgreSQL
* Docker (optional)

### Clone the Repository

```bash
git clone <repository-url>
cd OortObjectStorage
```

### Configure Environment

Create a `.env` file:

```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=masterpassword
DB_NAME=oort_objects

APP_PORT=3333
```

### Run the Server

```bash
go run cmd/server/main.go
```

### Run Tests

```bash
go test ./...
```

---

## Project Goals

This project is intended to explore:

* Object storage systems
* Streaming file uploads
* Storage engine design
* Database-backed metadata management
* Infrastructure software architecture
* Distributed systems concepts
