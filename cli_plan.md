# CLI Plan — `oortctl`

## Concepts

The CLI has two auth modes:

1. **Human (JWT):** `oortctl login` stores a JWT in `~/.oort/token`. Every command reads it automatically.
2. **Machine (API key):** Pass `--api-key` flag on any command — skips JWT entirely.

Config (optional) at `~/.oort/config.json`:
```json
{ "server": "http://localhost:3333" }
```

## Commands

```
oortctl login --email <email> --password <password>
  → POST /auth/login, stores JWT at ~/.oort/token

oortctl bucket create <name>
oortctl bucket list

oortctl upload <bucket> <file> [--key <key>]
  → PUT multipart to /buckets/{bucket}/objects/{key}
  → key defaults to file's basename

oortctl download <bucket> <key> [-o <output>]
  → GET /buckets/{bucket}/objects/{key}, writes to file or stdout

oortctl delete <bucket> <key>

oortctl ls <bucket>
  → GET /buckets/{bucket}/objects/

oortctl sign <bucket> <key> --op <upload|download> [--expires <duration>]
  → POST /buckets/{bucket}/objects/{key}/sign, prints URL

oortctl logout
  → rm ~/.oort/token
```

## Internal Client Package (`internal/client`)

Thin HTTP wrapper — one method per API endpoint:

```go
type Client struct {
    ServerURL string
    JWT       string  // optional
    APIKey    string  // optional
}

func (c *Client) Login(ctx, email, password) (*LoginResult, error)
func (c *Client) CreateBucket(ctx, name) (*Bucket, error)
func (c *Client) ListBuckets(ctx) ([]Bucket, error)
func (c *Client) UploadObject(ctx, bucket, key, contentType string, body io.Reader, size int64) (*Object, error)
func (c *Client) DownloadObject(ctx, bucket, key) (io.ReadCloser, *Object, error)
func (c *Client) DeleteObject(ctx, bucket, key) error
func (c *Client) ListObjects(ctx, bucket) ([]Object, error)
func (c *Client) SignURL(ctx, bucket, key, operation string, expires time.Duration) (*SignResult, error)
```

Each method:
- Builds URL from `ServerURL`
- Adds `Authorization: Bearer <JWT>` or `X-API-Key: <key>` header
- Returns parsed response or error

## Dependency

Only `github.com/spf13/cobra` — well-known, widely used, handles subcommands and flags.

## Files

```
cmd/oortctl/main.go            — root command, subcommands
internal/client/client.go      — HTTP client
internal/client/auth.go        — login + token storage helpers
```

## What it replaces

No more curl. A human workflow becomes:

```bash
oortctl login --email admin@oort.local --password admin123
oortctl bucket create artifacts
oortctl upload artifacts ./release.tar.gz
oortctl sign artifacts release.tar.gz --op download --expires 24h
# → prints signed URL, share with anyone
```

A machine workflow (CI/CD):

```bash
oortctl upload artifacts ./build.zip --api-key oort_xxx
```
