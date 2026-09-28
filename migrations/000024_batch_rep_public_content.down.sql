DELETE FROM role_permissions rp
USING permissions p
WHERE rp.permission_id = p.id
  AND p.code = 'public_content.contribute';
DELETE FROM permissions WHERE code = 'public_content.contribute';
