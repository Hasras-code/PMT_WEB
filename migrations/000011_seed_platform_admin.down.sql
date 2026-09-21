WITH platform_admin AS (
 SELECT
  u.id AS target_user_id,
  r.id AS platform_role_id
 FROM users u
 CROSS JOIN roles r
 WHERE lower(u.student_number)=lower('as123123')
   AND lower(u.email)=lower('test@example.com')
   AND r.code='PLATFORM_ADMIN'
   AND r.scope='PLATFORM'
   AND EXISTS (
    SELECT 1
    FROM audit_logs a
    WHERE a.action='PLATFORM_ROLE_ASSIGNED'
      AND a.entity_type='user'
      AND a.entity_id=u.id
      AND a.new_values @> '{"role":"PLATFORM_ADMIN","source":"migration_000011"}'::jsonb
   )
)
DELETE FROM user_platform_roles upr
USING platform_admin pa
WHERE upr.user_id=pa.target_user_id
  AND upr.role_id=pa.platform_role_id;
