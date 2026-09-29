DROP INDEX upload_intents_storage_cleanup;
ALTER TABLE upload_intents
 DROP COLUMN storage_class,
 DROP COLUMN storage_provider;
