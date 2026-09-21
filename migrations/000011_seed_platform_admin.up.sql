WITH platform_admin AS (
 SELECT
  u.id AS target_user_id,
  r.id AS platform_role_id,
  r.scope AS role_scope
 FROM users u
 CROSS JOIN roles r
 WHERE lower(u.student_number)=lower('as123123')
   AND lower(u.email)=lower('test@example.com')
   AND u.status='ACTIVE'
   AND r.code='PLATFORM_ADMIN'
   AND r.scope='PLATFORM'
), assigned AS (
 INSERT INTO user_platform_roles(user_id,role_id,scope,assigned_by)
 SELECT
  pa.target_user_id,
  pa.platform_role_id,
  pa.role_scope,
  NULL
 FROM platform_admin pa
 ON CONFLICT DO NOTHING
 RETURNING user_id
)
INSERT INTO audit_logs(actor_user_id,action,entity_type,entity_id,new_values)
SELECT NULL,'PLATFORM_ROLE_ASSIGNED','user',user_id,
 jsonb_build_object('role','PLATFORM_ADMIN','source','migration_000011')
FROM assigned;
