DROP TABLE fund_transaction_attachments;
ALTER TABLE upload_intents DROP CONSTRAINT upload_intents_purpose_check;
ALTER TABLE upload_intents ADD CONSTRAINT upload_intents_purpose_check CHECK(purpose IN ('resource','attachment','gallery','profile','event','batch'));
