CREATE TABLE IF NOT EXISTS signed_urls (
    id UUID PRIMARY KEY,
    bucket_name TEXT NOT NULL,
    object_key TEXT NOT NULL,
    token TEXT UNIQUE NOT NULL,
    operation TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
