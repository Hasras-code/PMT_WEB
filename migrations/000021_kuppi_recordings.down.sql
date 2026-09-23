DELETE FROM notifications WHERE event_id IN (SELECT id FROM notification_events WHERE entity_id IN (SELECT id FROM kuppi_recordings));
DELETE FROM notification_events WHERE entity_id IN (SELECT id FROM kuppi_recordings);
DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code LIKE 'kuppi.%');
DELETE FROM permissions WHERE code LIKE 'kuppi.%';
DROP INDEX IF EXISTS kuppi_recordings_active_video;
DROP INDEX IF EXISTS kuppi_recordings_module_status;
DROP INDEX IF EXISTS kuppi_recordings_batch_status;
ALTER TABLE kuppi_recordings DROP CONSTRAINT IF EXISTS kuppi_status_dates;
ALTER TABLE kuppi_recordings DROP CONSTRAINT IF EXISTS kuppi_description_length;
ALTER TABLE kuppi_recordings DROP CONSTRAINT IF EXISTS kuppi_title_length;
ALTER TABLE kuppi_recordings DROP COLUMN IF EXISTS archived_at;
ALTER TABLE kuppi_recordings DROP COLUMN IF EXISTS sort_order;
ALTER TABLE kuppi_recordings ALTER COLUMN recorded_at TYPE date USING recorded_at::date;
ALTER TABLE kuppi_recordings RENAME COLUMN recorded_at TO lesson_date;
ALTER TABLE kuppi_recordings RENAME TO recorded_lessons;
INSERT INTO permissions(code,scope) VALUES('lesson.view','BATCH'),('lesson.manage','BATCH');
INSERT INTO role_permissions
SELECT r.id,p.id,'BATCH' FROM roles r CROSS JOIN permissions p
WHERE r.code = 'STUDENT' AND p.code = 'lesson.view';
INSERT INTO role_permissions
SELECT r.id,p.id,'BATCH' FROM roles r CROSS JOIN permissions p
WHERE r.code IN ('BATCH_REP','CONTENT_MANAGER','ACADEMIC_REP') AND p.code = 'lesson.manage';
