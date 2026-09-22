ALTER TABLE users
 ADD COLUMN registration_batch_id uuid REFERENCES batches(id);

CREATE INDEX users_registration_batch
 ON users(registration_batch_id)
 WHERE registration_batch_id IS NOT NULL;

INSERT INTO role_permissions(role_id,permission_id,scope)
SELECT r.id,p.id,'BATCH'
FROM roles r
JOIN permissions p ON p.code='role.assign' AND p.scope='BATCH'
WHERE r.code='ACADEMIC_REP' AND r.scope='BATCH'
ON CONFLICT DO NOTHING;
