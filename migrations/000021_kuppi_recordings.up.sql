DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM recorded_lessons WHERE title = '' OR length(title) > 200 OR btrim(title) = '') THEN
        RAISE EXCEPTION 'recorded_lessons contains an invalid title';
    END IF;
    IF EXISTS (SELECT 1 FROM recorded_lessons WHERE youtube_video_id !~ '^[A-Za-z0-9_-]{11}$') THEN
        RAISE EXCEPTION 'recorded_lessons contains an invalid YouTube video ID';
    END IF;
    IF EXISTS (SELECT 1 FROM recorded_lessons l LEFT JOIN modules m ON m.id = l.module_id AND m.batch_id = l.batch_id WHERE m.id IS NULL) THEN
        RAISE EXCEPTION 'recorded_lessons contains an invalid module relationship';
    END IF;
    IF EXISTS (SELECT 1 FROM recorded_lessons WHERE status <> 'ARCHIVED' GROUP BY batch_id, module_id, youtube_video_id HAVING count(*) > 1) THEN
        RAISE EXCEPTION 'recorded_lessons contains duplicate active videos';
    END IF;
END $$;

ALTER TABLE recorded_lessons RENAME TO kuppi_recordings;
ALTER TABLE kuppi_recordings RENAME COLUMN lesson_date TO recorded_at;
ALTER TABLE kuppi_recordings ALTER COLUMN recorded_at TYPE timestamptz USING recorded_at::timestamp AT TIME ZONE 'Asia/Colombo';
ALTER TABLE kuppi_recordings ADD COLUMN sort_order integer NOT NULL DEFAULT 0;
ALTER TABLE kuppi_recordings ADD COLUMN archived_at timestamptz;
UPDATE kuppi_recordings SET archived_at = COALESCE(updated_at, now()) WHERE status = 'ARCHIVED';
ALTER TABLE kuppi_recordings ADD CONSTRAINT kuppi_title_length CHECK (length(btrim(title)) BETWEEN 1 AND 200);
ALTER TABLE kuppi_recordings ADD CONSTRAINT kuppi_description_length CHECK (length(description) <= 20000);
ALTER TABLE kuppi_recordings ADD CONSTRAINT kuppi_status_dates CHECK ((status <> 'PUBLISHED' OR published_at IS NOT NULL) AND (status <> 'ARCHIVED' OR archived_at IS NOT NULL));
CREATE INDEX kuppi_recordings_batch_status ON kuppi_recordings(batch_id,status,published_at DESC,id DESC);
CREATE INDEX kuppi_recordings_module_status ON kuppi_recordings(batch_id,module_id,status);
CREATE UNIQUE INDEX kuppi_recordings_active_video ON kuppi_recordings(batch_id,module_id,youtube_video_id) WHERE status <> 'ARCHIVED';

INSERT INTO permissions(code,scope) VALUES
    ('kuppi.view','BATCH'),
    ('kuppi.create','BATCH'),
    ('kuppi.update','BATCH'),
    ('kuppi.publish','BATCH'),
    ('kuppi.archive','BATCH')
ON CONFLICT (code) DO NOTHING;
INSERT INTO role_permissions
SELECT r.id,p.id,'BATCH' FROM roles r CROSS JOIN permissions p
WHERE r.code IN ('BATCH_REP','ACADEMIC_REP') AND p.code LIKE 'kuppi.%';
INSERT INTO role_permissions
SELECT r.id,p.id,'BATCH' FROM roles r CROSS JOIN permissions p
WHERE r.code = 'STUDENT' AND p.code = 'kuppi.view';

DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code IN ('lesson.view','lesson.manage'));
DELETE FROM permissions WHERE code IN ('lesson.view','lesson.manage');
