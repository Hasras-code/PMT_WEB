CREATE TABLE fund_managers(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), batch_id uuid NOT NULL, fund_id uuid NOT NULL, membership_id uuid NOT NULL,
 assigned_by uuid NOT NULL REFERENCES users(id), assigned_at timestamptz NOT NULL DEFAULT now(),
 revoked_at timestamptz, revoked_by uuid REFERENCES users(id),
 UNIQUE(batch_id,id), FOREIGN KEY(batch_id,fund_id) REFERENCES funds(batch_id,id),
 FOREIGN KEY(batch_id,membership_id) REFERENCES batch_memberships(batch_id,id)
);
CREATE UNIQUE INDEX fund_managers_active ON fund_managers(fund_id,membership_id) WHERE revoked_at IS NULL;
CREATE INDEX fund_managers_lookup ON fund_managers(batch_id,membership_id,fund_id) WHERE revoked_at IS NULL;
