-- 0002_icons_and_labels (down): remove the added columns.

ALTER TABLE bookmarks  DROP COLUMN tags;
ALTER TABLE bookmarks  DROP COLUMN color;
ALTER TABLE categories DROP COLUMN color;
ALTER TABLE categories DROP COLUMN icon;
