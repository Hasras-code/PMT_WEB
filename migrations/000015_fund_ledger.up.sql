CREATE TABLE fund_transactions(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), batch_id uuid NOT NULL, fund_id uuid NOT NULL,
 type text NOT NULL CHECK(type IN ('CASH_IN','EXPENSE','TRANSFER_IN','TRANSFER_OUT','REVERSAL_IN','REVERSAL_OUT')),
 amount_minor bigint NOT NULL CHECK(amount_minor>0), description text NOT NULL CHECK(length(description)<=20000),
 category text, source_type text, source_name text, reference text,
 transaction_date timestamptz NOT NULL DEFAULT now(), created_by uuid NOT NULL REFERENCES users(id),
 status text NOT NULL DEFAULT 'DRAFT' CHECK(status IN ('DRAFT','POSTED')),
 reversal_of_transaction_id uuid, reversed_by_transaction_id uuid,
 created_at timestamptz NOT NULL DEFAULT now(), posted_at timestamptz,
 UNIQUE(batch_id,id), UNIQUE(batch_id,fund_id,id),
 FOREIGN KEY(batch_id,fund_id) REFERENCES funds(batch_id,id),
 FOREIGN KEY(batch_id,reversal_of_transaction_id) REFERENCES fund_transactions(batch_id,id),
 FOREIGN KEY(batch_id,reversed_by_transaction_id) REFERENCES fund_transactions(batch_id,id),
 CHECK((status='POSTED' AND posted_at IS NOT NULL) OR (status='DRAFT' AND posted_at IS NULL))
);
CREATE UNIQUE INDEX fund_transactions_one_reversal ON fund_transactions(reversal_of_transaction_id) WHERE reversal_of_transaction_id IS NOT NULL;
CREATE INDEX fund_transactions_history ON fund_transactions(batch_id,fund_id,transaction_date DESC,id DESC);
CREATE INDEX fund_transactions_balance ON fund_transactions(fund_id,type) WHERE status='POSTED';
CREATE FUNCTION protect_posted_fund_transaction() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' AND OLD.status='POSTED' THEN RAISE EXCEPTION 'posted fund transactions are immutable' USING ERRCODE='23514'; END IF;
 IF TG_OP='UPDATE' AND OLD.status='POSTED' AND
   (NEW.batch_id,NEW.fund_id,NEW.type,NEW.amount_minor,NEW.transaction_date,NEW.created_by,COALESCE(NEW.reversal_of_transaction_id,'00000000-0000-0000-0000-000000000000')) IS DISTINCT FROM
   (OLD.batch_id,OLD.fund_id,OLD.type,OLD.amount_minor,OLD.transaction_date,OLD.created_by,COALESCE(OLD.reversal_of_transaction_id,'00000000-0000-0000-0000-000000000000'))
 THEN RAISE EXCEPTION 'posted fund transactions are immutable' USING ERRCODE='23514'; END IF;
 RETURN COALESCE(NEW,OLD);
END $$;
CREATE TRIGGER fund_transactions_immutable BEFORE UPDATE OR DELETE ON fund_transactions FOR EACH ROW EXECUTE FUNCTION protect_posted_fund_transaction();
