DELETE FROM role_permissions rp
USING roles r,permissions p
WHERE rp.role_id=r.id
  AND rp.permission_id=p.id
  AND r.code='ACADEMIC_REP'
  AND p.code='role.assign'
  AND rp.scope='BATCH';

DROP INDEX users_registration_batch;
ALTER TABLE users DROP COLUMN registration_batch_id;
