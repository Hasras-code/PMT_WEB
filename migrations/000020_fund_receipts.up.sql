ALTER TABLE upload_intents DROP CONSTRAINT upload_intents_purpose_check;
ALTER TABLE upload_intents ADD CONSTRAINT upload_intents_purpose_check CHECK(purpose IN ('resource','attachment','gallery','profile','event','batch','fund_receipt'));
CREATE TABLE fund_transaction_attachments(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), batch_id uuid NOT NULL, fund_id uuid NOT NULL, transaction_id uuid NOT NULL,
 file_name text NOT NULL, storage_key text NOT NULL UNIQUE, mime_type text NOT NULL CHECK(mime_type='application/pdf'), size_bytes bigint NOT NULL CHECK(size_bytes>0),
 uploaded_by uuid NOT NULL REFERENCES users(id), visibility text NOT NULL DEFAULT 'MANAGERS' CHECK(visibility IN ('MEMBERS','MANAGERS')), created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(batch_id,id), FOREIGN KEY(batch_id,fund_id,transaction_id) REFERENCES fund_transactions(batch_id,fund_id,id)
);
CREATE INDEX fund_attachments_transaction ON fund_transaction_attachments(batch_id,fund_id,transaction_id,created_at,id);
