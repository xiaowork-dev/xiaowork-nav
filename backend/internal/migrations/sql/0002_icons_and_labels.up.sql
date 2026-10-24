-- 0002_icons_and_labels: extend the data model for richer icons and labels.
--
-- Adds (all non-destructive, with defaults so existing rows stay valid):
--   categories.icon   - emoji / short text icon for the category
--   categories.color  - optional accent colour (hex)
--   bookmarks.color   - optional accent colour (hex)
--   bookmarks.tags    - comma-separated labels (legacy `tag` values)
--
-- bookmark.icon_type gains new values (no schema change required):
--   favicon | simple | text

ALTER TABLE categories ADD COLUMN icon  TEXT NOT NULL DEFAULT '';
ALTER TABLE categories ADD COLUMN color TEXT NOT NULL DEFAULT '';

ALTER TABLE bookmarks ADD COLUMN color TEXT NOT NULL DEFAULT '';
ALTER TABLE bookmarks ADD COLUMN tags  TEXT NOT NULL DEFAULT '';
