-- 0003_icon_metadata: metadata for automatically fetched and cached icons.
--
-- All columns are additive with defaults, so existing rows stay valid and an
-- older image can still read the database (columns are never dropped).
--
--   bookmarks.icon_source     - how the current icon was obtained:
--                               '' | manual | link-icon | apple-touch | manifest
--                               | favicon-ico | brand-guess
--   bookmarks.icon_status     - outcome of the last automatic attempt: '' | ok | failed
--   bookmarks.icon_error      - human-readable reason for the last failure
--   bookmarks.icon_checked_at - RFC 3339 UTC timestamp of the last automatic attempt
--
--   uploads.source_url        - remote URL an auto-cached icon was fetched from
--                               (empty for user uploads)
--   uploads.origin            - 'manual' (user upload) | 'auto' (server fetch)

ALTER TABLE bookmarks ADD COLUMN icon_source     TEXT NOT NULL DEFAULT '';
ALTER TABLE bookmarks ADD COLUMN icon_status     TEXT NOT NULL DEFAULT '';
ALTER TABLE bookmarks ADD COLUMN icon_error      TEXT NOT NULL DEFAULT '';
ALTER TABLE bookmarks ADD COLUMN icon_checked_at TEXT NOT NULL DEFAULT '';

ALTER TABLE uploads ADD COLUMN source_url TEXT NOT NULL DEFAULT '';
ALTER TABLE uploads ADD COLUMN origin     TEXT NOT NULL DEFAULT 'manual';

-- Cache lookups deduplicate by source URL, so several bookmarks can share one
-- cached file. The index keeps that lookup cheap on large instances.
CREATE INDEX IF NOT EXISTS idx_uploads_source_url ON uploads (source_url);

-- Icon bookkeeping is queried per status during bulk completion.
CREATE INDEX IF NOT EXISTS idx_bookmarks_icon_status ON bookmarks (icon_status);
