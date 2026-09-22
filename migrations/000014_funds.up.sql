INSERT INTO permissions(code,scope) VALUES
 ('fund.view','BATCH'),('fund.create','BATCH'),('fund.update','BATCH'),('fund.close','BATCH'),
 ('fund.transaction.create','BATCH'),('fund.transaction.reverse','BATCH'),
 ('fund.manager.assign','BATCH'),('fund.manager.remove','BATCH'),
 ('fund.transfer.create','BATCH'),('fund.transfer.reverse','BATCH'),
 ('birthday_fund.manage','BATCH'),('birthday_contribution.record','BATCH')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions(role_id,permission_id,scope)
SELECT r.id,p.id,'BATCH' FROM roles r JOIN permissions p ON p.scope='BATCH'
 AND p.code = ANY(ARRAY['fund.view'])
WHERE r.code='STUDENT' AND r.scope='BATCH' ON CONFLICT DO NOTHING;

INSERT INTO role_permissions(role_id,permission_id,scope)
SELECT r.id,p.id,'BATCH' FROM roles r JOIN permissions p ON p.scope='BATCH'
 AND p.code = ANY(ARRAY['fund.view','fund.create','fund.update','fund.close','fund.transaction.create','fund.transaction.reverse','fund.manager.assign','fund.manager.remove','fund.transfer.create','fund.transfer.reverse','birthday_fund.manage','birthday_contribution.record'])
WHERE r.code='BATCH_REP' AND r.scope='BATCH' ON CONFLICT DO NOTHING;

ALTER TABLE events ADD CONSTRAINT events_batch_id_id_unique UNIQUE(batch_id,id);

CREATE TABLE funds(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 batch_id uuid NOT NULL REFERENCES batches(id),
 name text NOT NULL CHECK(length(btrim(name)) BETWEEN 1 AND 200),
 description text NOT NULL DEFAULT '' CHECK(length(description)<=20000),
 type text NOT NULL CHECK(type IN ('BIRTHDAY','EVENT','OTHER')),
 event_id uuid,
 currency text NOT NULL DEFAULT 'LKR' CHECK(currency='LKR'),
 status text NOT NULL DEFAULT 'ACTIVE' CHECK(status IN ('ACTIVE','CLOSED','ARCHIVED')),
 created_by uuid REFERENCES users(id),
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
 closed_at timestamptz, closed_by uuid REFERENCES users(id),
 UNIQUE(batch_id,id),
 FOREIGN KEY(batch_id,event_id) REFERENCES events(batch_id,id),
 CHECK((type='EVENT') OR event_id IS NULL),
 CHECK((status='ACTIVE' AND closed_at IS NULL AND closed_by IS NULL) OR status<>'ACTIVE')
);
CREATE UNIQUE INDEX funds_one_birthday ON funds(batch_id) WHERE type='BIRTHDAY';
CREATE INDEX funds_list ON funds(batch_id,status,type,created_at DESC,id DESC);

INSERT INTO funds(batch_id,name,type,created_by)
SELECT id,'Birthday Fund','BIRTHDAY',NULL FROM batches
ON CONFLICT (batch_id) WHERE type='BIRTHDAY' DO NOTHING;
