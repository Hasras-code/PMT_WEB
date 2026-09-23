DELETE FROM role_permissions
WHERE role_id IN (SELECT id FROM roles WHERE code IN ('BATCH_REP', 'ACADEMIC_REP') AND scope = 'BATCH')
  AND permission_id IN (SELECT id FROM permissions WHERE code IN ('kuppi.view', 'kuppi.create', 'kuppi.update', 'kuppi.publish', 'kuppi.archive') AND scope = 'BATCH');
