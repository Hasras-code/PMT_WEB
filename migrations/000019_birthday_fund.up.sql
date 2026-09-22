CREATE TABLE birthday_contribution_periods(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), batch_id uuid NOT NULL, fund_id uuid NOT NULL,
 year int NOT NULL CHECK(year BETWEEN 2000 AND 2200), month int NOT NULL CHECK(month BETWEEN 1 AND 12),
 amount_minor bigint NOT NULL CHECK(amount_minor>0), status text NOT NULL DEFAULT 'OPEN' CHECK(status IN ('OPEN','CLOSED')),
 created_by uuid NOT NULL REFERENCES users(id), created_at timestamptz NOT NULL DEFAULT now(), closed_at timestamptz, closed_by uuid REFERENCES users(id),
 UNIQUE(batch_id,id), UNIQUE(batch_id,year,month), FOREIGN KEY(batch_id,fund_id) REFERENCES funds(batch_id,id)
);
CREATE TABLE birthday_contributions(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), batch_id uuid NOT NULL, period_id uuid NOT NULL, membership_id uuid NOT NULL,
 expected_minor bigint NOT NULL CHECK(expected_minor>0), created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(batch_id,id), UNIQUE(period_id,membership_id),
 FOREIGN KEY(batch_id,period_id) REFERENCES birthday_contribution_periods(batch_id,id),
 FOREIGN KEY(batch_id,membership_id) REFERENCES batch_memberships(batch_id,id)
);
CREATE TABLE birthday_contribution_payments(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), batch_id uuid NOT NULL, contribution_id uuid NOT NULL,
 transaction_id uuid NOT NULL, amount_minor bigint NOT NULL CHECK(amount_minor>0), recorded_by uuid NOT NULL REFERENCES users(id), created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(batch_id,id), UNIQUE(transaction_id),
 FOREIGN KEY(batch_id,contribution_id) REFERENCES birthday_contributions(batch_id,id),
 FOREIGN KEY(batch_id,transaction_id) REFERENCES fund_transactions(batch_id,id)
);
CREATE INDEX birthday_contributions_period ON birthday_contributions(batch_id,period_id);
CREATE INDEX birthday_payments_contribution ON birthday_contribution_payments(contribution_id);

CREATE FUNCTION validate_birthday_finance() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE valid boolean;
BEGIN
 IF TG_TABLE_NAME='birthday_contribution_periods' THEN
  SELECT EXISTS(SELECT 1 FROM funds f WHERE f.id=NEW.fund_id AND f.batch_id=NEW.batch_id AND f.type='BIRTHDAY') INTO valid;
  IF NOT valid THEN RAISE EXCEPTION 'birthday period requires the batch Birthday Fund' USING ERRCODE='23514'; END IF;
 ELSE
  SELECT EXISTS(
   SELECT 1 FROM birthday_contributions c
   JOIN birthday_contribution_periods p ON p.id=c.period_id AND p.batch_id=c.batch_id
   JOIN fund_transactions t ON t.id=NEW.transaction_id AND t.batch_id=NEW.batch_id
   WHERE c.id=NEW.contribution_id AND c.batch_id=NEW.batch_id AND t.fund_id=p.fund_id
    AND t.type='CASH_IN' AND t.status='POSTED' AND t.source_type='STUDENT_CONTRIBUTION'
    AND t.amount_minor=NEW.amount_minor
  ) INTO valid;
  IF NOT valid THEN RAISE EXCEPTION 'birthday payment requires its matching posted ledger entry' USING ERRCODE='23514'; END IF;
 END IF;
 RETURN NEW;
END $$;
CREATE CONSTRAINT TRIGGER birthday_period_fund_valid AFTER INSERT OR UPDATE ON birthday_contribution_periods DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION validate_birthday_finance();
CREATE CONSTRAINT TRIGGER birthday_payment_ledger_valid AFTER INSERT OR UPDATE ON birthday_contribution_payments DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION validate_birthday_finance();

CREATE FUNCTION protect_birthday_payment() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'birthday payments are immutable' USING ERRCODE='23514'; END $$;
CREATE TRIGGER birthday_payments_immutable BEFORE UPDATE OR DELETE ON birthday_contribution_payments FOR EACH ROW EXECUTE FUNCTION protect_birthday_payment();
