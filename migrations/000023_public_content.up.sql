INSERT INTO permissions(code, scope)
VALUES ('public_content.manage', 'PLATFORM')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions(role_id, permission_id, scope)
SELECT r.id, p.id, 'PLATFORM'
FROM roles r
JOIN permissions p ON p.code = 'public_content.manage' AND p.scope = 'PLATFORM'
WHERE r.code = 'PLATFORM_ADMIN' AND r.scope = 'PLATFORM'
ON CONFLICT (role_id, permission_id) DO NOTHING;

CREATE TABLE public_hero_slides (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 title text NOT NULL CHECK(length(title) BETWEEN 1 AND 200),
 subtitle text NOT NULL DEFAULT '' CHECK(length(subtitle) <= 500),
 cta_text text NOT NULL DEFAULT '' CHECK(length(cta_text) <= 100),
 image_url text NOT NULL CHECK(length(image_url) BETWEEN 1 AND 2048),
 priority int NOT NULL DEFAULT 0 CHECK(priority BETWEEN -10000 AND 10000),
 created_by uuid NOT NULL REFERENCES users(id),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 deleted_at timestamptz
);
CREATE INDEX public_hero_slides_order ON public_hero_slides(priority, created_at DESC, id) WHERE deleted_at IS NULL;

CREATE TABLE public_events (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 title text NOT NULL CHECK(length(title) BETWEEN 1 AND 300),
 event_date date NOT NULL,
 image_url text NOT NULL CHECK(length(image_url) BETWEEN 1 AND 2048),
 external_url text NOT NULL DEFAULT '' CHECK(length(external_url) <= 2048),
 created_by uuid NOT NULL REFERENCES users(id),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 deleted_at timestamptz
);
CREATE INDEX public_events_date ON public_events(event_date DESC, id DESC) WHERE deleted_at IS NULL;

CREATE TABLE public_gallery_items (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 title text NOT NULL CHECK(length(title) BETWEEN 1 AND 300),
 category text NOT NULL CHECK(category IN ('Academic', 'Workshops', 'Social', 'Sports')),
 image_url text NOT NULL CHECK(length(image_url) BETWEEN 1 AND 2048),
 visibility text NOT NULL CHECK(visibility IN ('Public', 'Internal')),
 created_by uuid NOT NULL REFERENCES users(id),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 deleted_at timestamptz
);
CREATE INDEX public_gallery_items_recent ON public_gallery_items(created_at DESC, id DESC) WHERE deleted_at IS NULL;

CREATE TABLE public_achievements (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 title text NOT NULL CHECK(length(title) BETWEEN 1 AND 100),
 count int NOT NULL CHECK(count BETWEEN 0 AND 1000000000),
 icon text NOT NULL DEFAULT 'TrophyIcon' CHECK(length(icon) <= 100),
 created_by uuid NOT NULL REFERENCES users(id),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 deleted_at timestamptz
);
CREATE INDEX public_achievements_order ON public_achievements(created_at, id) WHERE deleted_at IS NULL;

CREATE TABLE public_social_links (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 singleton boolean NOT NULL DEFAULT true UNIQUE CHECK(singleton),
 whatsapp text NOT NULL DEFAULT '' CHECK(length(whatsapp) <= 2048),
 facebook text NOT NULL DEFAULT '' CHECK(length(facebook) <= 2048),
 instagram text NOT NULL DEFAULT '' CHECK(length(instagram) <= 2048),
 youtube text NOT NULL DEFAULT '' CHECK(length(youtube) <= 2048),
 updated_by uuid NOT NULL REFERENCES users(id),
 updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE public_featured_representatives (
 assignment_id uuid PRIMARY KEY REFERENCES position_assignments(id) ON DELETE CASCADE,
 featured_by uuid NOT NULL REFERENCES users(id),
 featured_at timestamptz NOT NULL DEFAULT now()
);
