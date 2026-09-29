ALTER TABLE upload_intents
 ADD COLUMN storage_provider text NOT NULL DEFAULT 'LOCAL'
  CHECK(storage_provider IN ('LOCAL','R2')),
 ADD COLUMN storage_class text NOT NULL DEFAULT 'PRIVATE'
  CHECK(storage_class IN ('PRIVATE','PUBLIC'));

UPDATE upload_intents
SET storage_class='PUBLIC'
WHERE purpose IN ('gallery','batch');

CREATE INDEX upload_intents_storage_cleanup
 ON upload_intents(storage_provider,storage_class,state,expires_at);
