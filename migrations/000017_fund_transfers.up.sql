CREATE TABLE fund_transfers(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), batch_id uuid NOT NULL,
 from_fund_id uuid NOT NULL, to_fund_id uuid NOT NULL,
 type text NOT NULL CHECK(type IN ('TRANSFER','LOAN','LOAN_REPAYMENT','REVERSAL')),
 amount_minor bigint NOT NULL CHECK(amount_minor>0), description text NOT NULL DEFAULT '' CHECK(length(description)<=20000),
 created_by uuid NOT NULL REFERENCES users(id), parent_transfer_id uuid,
 reversal_of_transfer_id uuid, reversed_by_transfer_id uuid,
 created_at timestamptz NOT NULL DEFAULT now(), reversed_at timestamptz, reversed_by uuid REFERENCES users(id),
 UNIQUE(batch_id,id), CHECK(from_fund_id<>to_fund_id),
 FOREIGN KEY(batch_id,from_fund_id) REFERENCES funds(batch_id,id), FOREIGN KEY(batch_id,to_fund_id) REFERENCES funds(batch_id,id),
 FOREIGN KEY(batch_id,parent_transfer_id) REFERENCES fund_transfers(batch_id,id),
 FOREIGN KEY(batch_id,reversal_of_transfer_id) REFERENCES fund_transfers(batch_id,id),
 FOREIGN KEY(batch_id,reversed_by_transfer_id) REFERENCES fund_transfers(batch_id,id),
 CHECK((type='LOAN_REPAYMENT' AND parent_transfer_id IS NOT NULL) OR (type<>'LOAN_REPAYMENT')),
 CHECK((type='REVERSAL' AND reversal_of_transfer_id IS NOT NULL) OR (type<>'REVERSAL'))
);
CREATE UNIQUE INDEX fund_transfers_one_reversal ON fund_transfers(reversal_of_transfer_id) WHERE reversal_of_transfer_id IS NOT NULL;
CREATE INDEX fund_transfers_history ON fund_transfers(batch_id,created_at DESC,id DESC);
CREATE INDEX fund_transfers_repayments ON fund_transfers(parent_transfer_id) WHERE type='LOAN_REPAYMENT';
ALTER TABLE fund_transactions ADD COLUMN transfer_id uuid;
ALTER TABLE fund_transactions ADD CONSTRAINT fund_transactions_transfer_fk FOREIGN KEY(batch_id,transfer_id) REFERENCES fund_transfers(batch_id,id);
CREATE INDEX fund_transactions_transfer ON fund_transactions(transfer_id) WHERE transfer_id IS NOT NULL;
