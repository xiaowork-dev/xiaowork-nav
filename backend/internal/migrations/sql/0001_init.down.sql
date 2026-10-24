-- 0001_init (down): drop all tables created by the initial migration.

DROP TABLE IF EXISTS uploads;
DROP TABLE IF EXISTS site_settings;
DROP TABLE IF EXISTS bookmarks;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
