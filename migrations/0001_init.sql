CREATE TABLE IF NOT EXISTS buckets (
    id UUID PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS objects (
    id UUID PRIMARY KEY,
    bucket_id UUID REFERENCES buckets(id),
    object_key TEXT NOT NULL,
    storage_path TEXT NOT NULL,
    content_type TEXT,
    size_bytes BIGINT,
    checksum TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);