INSERT INTO role_permissions(role_id, permission_id, scope)
SELECT r.id, p.id, 'BATCH'
FROM roles r
CROSS JOIN permissions p
WHERE r.code IN ('BATCH_REP', 'ACADEMIC_REP')
  AND r.scope = 'BATCH'
  AND p.code IN ('kuppi.view', 'kuppi.create', 'kuppi.update', 'kuppi.publish', 'kuppi.archive')
  AND p.scope = 'BATCH'
ON CONFLICT (role_id, permission_id) DO NOTHING;
