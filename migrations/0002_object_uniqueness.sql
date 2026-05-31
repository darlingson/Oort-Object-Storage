CREATE UNIQUE INDEX IF NOT EXISTS idx_object_bucket_key
ON objects(bucket_id, object_key);