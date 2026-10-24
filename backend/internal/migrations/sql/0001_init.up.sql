-- 0001_init: initial schema for XiaoWork Nav V2.0
-- Tables: users, sessions, categories, bookmarks, site_settings, uploads.
-- schema_migrations is created by the migration runner, not here.

CREATE TABLE IF NOT EXISTS users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT    NOT NULL UNIQUE,
    password_hash TEXT    NOT NULL,
    created_at    TEXT    NOT NULL,
    updated_at    TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
    id           TEXT    PRIMARY KEY,
    user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at   TEXT    NOT NULL,
    expires_at   TEXT    NOT NULL,
    last_seen_at TEXT    NOT NULL,
    user_agent   TEXT    NOT NULL DEFAULT '',
    ip           TEXT    NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);

CREATE TABLE IF NOT EXISTS categories (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT    NOT NULL,
    description TEXT    NOT NULL DEFAULT '',
    sort_order  INTEGER NOT NULL DEFAULT 0,
    is_visible  INTEGER NOT NULL DEFAULT 1,
    created_at  TEXT    NOT NULL,
    updated_at  TEXT    NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_categories_sort_order ON categories(sort_order);
CREATE INDEX IF NOT EXISTS idx_categories_is_visible ON categories(is_visible);

CREATE TABLE IF NOT EXISTS bookmarks (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    category_id     INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    title           TEXT    NOT NULL,
    url             TEXT    NOT NULL,
    description     TEXT    NOT NULL DEFAULT '',
    icon_type       TEXT    NOT NULL DEFAULT 'none',
    icon_value      TEXT    NOT NULL DEFAULT '',
    keywords        TEXT    NOT NULL DEFAULT '',
    sort_order      INTEGER NOT NULL DEFAULT 0,
    is_visible      INTEGER NOT NULL DEFAULT 1,
    open_in_new_tab INTEGER NOT NULL DEFAULT 1,
    created_at      TEXT    NOT NULL,
    updated_at      TEXT    NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_bookmarks_category_id ON bookmarks(category_id);
CREATE INDEX IF NOT EXISTS idx_bookmarks_sort_order ON bookmarks(sort_order);
CREATE INDEX IF NOT EXISTS idx_bookmarks_is_visible ON bookmarks(is_visible);

CREATE TABLE IF NOT EXISTS site_settings (
    key        TEXT PRIMARY KEY,
    value_json TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS uploads (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    filename      TEXT    NOT NULL UNIQUE,
    original_name TEXT    NOT NULL DEFAULT '',
    mime          TEXT    NOT NULL,
    size          INTEGER NOT NULL,
    created_at    TEXT    NOT NULL
);
