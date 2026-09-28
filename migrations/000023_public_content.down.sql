DROP TABLE IF EXISTS public_featured_representatives;
DROP TABLE IF EXISTS public_social_links;
DROP TABLE IF EXISTS public_achievements;
DROP TABLE IF EXISTS public_gallery_items;
DROP TABLE IF EXISTS public_events;
DROP TABLE IF EXISTS public_hero_slides;

DELETE FROM role_permissions rp
USING permissions p
WHERE rp.permission_id = p.id
  AND p.code = 'public_content.manage';
DELETE FROM permissions WHERE code = 'public_content.manage';
