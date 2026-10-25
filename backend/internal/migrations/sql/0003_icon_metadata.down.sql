-- 0003_icon_metadata (down)
--
-- Kept for documentation and manual rollback only: the migration runner never
-- reads .down.sql files. Rolling back requires restoring a pre-update snapshot
-- (an older image cannot read a newer schema; see docs/updating.md §4).

DROP INDEX IF EXISTS idx_bookmarks_icon_status;
DROP INDEX IF EXISTS idx_uploads_source_url;

ALTER TABLE uploads   DROP COLUMN origin;
ALTER TABLE uploads   DROP COLUMN source_url;

ALTER TABLE bookmarks DROP COLUMN icon_checked_at;
ALTER TABLE bookmarks DROP COLUMN icon_error;
ALTER TABLE bookmarks DROP COLUMN icon_status;
ALTER TABLE bookmarks DROP COLUMN icon_source;
