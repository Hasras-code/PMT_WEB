INSERT INTO permissions(code, scope)
VALUES ('public_content.contribute', 'BATCH')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions(role_id, permission_id, scope)
SELECT r.id, p.id, 'BATCH'
FROM roles r
JOIN permissions p ON p.code = 'public_content.contribute' AND p.scope = 'BATCH'
WHERE r.code = 'BATCH_REP' AND r.scope = 'BATCH'
ON CONFLICT (role_id, permission_id) DO NOTHING;
