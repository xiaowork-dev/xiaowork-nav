/**
 * API types mirroring `docs/openapi/openapi.yaml`.
 *
 * Hand-maintained. Once the contract is frozen these can be regenerated with
 * `npm run gen:types` (openapi-typescript); this file then becomes a thin
 * re-export.
 */

export type Theme = 'light' | 'dark' | 'auto'
export type IconType = 'none' | 'url' | 'upload' | 'auto' | 'favicon' | 'simple' | 'text'
export type IconSource = 'manual' | 'link-icon' | 'apple-touch' | 'manifest' | 'favicon-ico' | 'brand-guess'
export type IconStatus = 'ok' | 'failed'
export type HomepageLayout = 'scroll' | 'tabs'
export type ConflictStrategy = 'skip' | 'overwrite' | 'rename'

export interface HealthResponse {
  status: string
  version?: string
}

export interface VersionInfo {
  version: string
  commit: string
  build_time: string
  schema_version: number
}

export interface User {
  id: number
  username: string
  created_at?: string
  updated_at?: string
}

export interface AuthStatus {
  initialized: boolean
  authenticated: boolean
  user?: User
}

export interface Category {
  id: number
  name: string
  description: string
  icon: string
  color: string
  sort_order: number
  is_visible: boolean
  bookmark_count?: number
  created_at?: string
  updated_at?: string
  bookmarks?: Bookmark[]
}

export interface CategoryInput {
  name: string
  description?: string
  icon?: string
  color?: string
  sort_order?: number
  is_visible?: boolean
}

export interface Bookmark {
  id: number
  category_id: number
  title: string
  url: string
  description: string
  icon_type: IconType
  icon_value: string
  keywords: string
  tags: string
  color: string
  sort_order: number
  is_visible: boolean
  open_in_new_tab: boolean
  /** How an automatically fetched icon was obtained (absent when not fetched). */
  icon_source?: IconSource
  icon_status?: IconStatus
  icon_error?: string
  icon_checked_at?: string
  created_at?: string
  updated_at?: string
}

export interface BookmarkInput {
  category_id: number
  title: string
  url: string
  description?: string
  icon_type?: IconType
  icon_value?: string
  keywords?: string
  tags?: string
  color?: string
  sort_order?: number
  is_visible?: boolean
  open_in_new_tab?: boolean
}

export interface SearchEngine {
  name: string
  url_template: string
}

export interface PublicSettings {
  site_title: string
  site_favicon: string
  theme: Theme
  background?: Record<string, unknown>
  search_engines: SearchEngine[]
  default_search_engine: string
  open_in_new_tab_default: boolean
  homepage_layout: HomepageLayout
}

export interface AdminSettings extends PublicSettings {
  allow_guest_access: boolean
}

export interface NavigationResponse {
  categories: Category[]
  settings: PublicSettings
}

export interface ListMeta {
  total: number
  page?: number
  page_size?: number
}

export interface Paginated<T> extends ListMeta {
  items: T[]
}

export interface OrderItem {
  id: number
  sort_order: number
}

export interface OrderResult {
  ok: boolean
  updated?: number
}

export interface UploadResult {
  filename: string
  url: string
  size: number
  mime: string
}

export interface IconRef {
  icon_type: IconType
  icon_value: string
}

/** Result of fetching an icon for one bookmark. */
export interface IconFillOutcome {
  bookmark?: Bookmark
  ok: boolean
  skipped?: boolean
  icon_source?: IconSource
  icon_error?: string
}

/** Icon coverage counts shown in the admin toolbar. */
export interface IconSummary {
  total: number
  manual: number
  auto: number
  favicon: number
  missing: number
  failed: number
  other?: number
}

export type IconJobStatus = 'running' | 'done' | 'canceled'

export interface IconJobItem {
  bookmark_id: number
  title?: string
  ok: boolean
  icon_source?: IconSource
  icon_error?: string
}

export interface IconJobSnapshot {
  id: string
  status: IconJobStatus
  total: number
  done: number
  succeeded: number
  failed: number
  skipped: number
  started_at: string
  finished_at?: string
  errors?: IconJobItem[]
}

export interface BulkIconRequest {
  only_missing?: boolean
  force?: boolean
  limit?: number
  refresh_days?: number
}

export interface Backup {
  id: string
  created_at: string
  app_version?: string
  schema_version?: number
  size?: number
}

export interface ImportPreview {
  format: string
  categories_new: number
  categories_conflict: number
  bookmarks_new: number
  bookmarks_conflict: number
  bookmarks_invalid?: number
  icons_kept?: number
  icons_converted?: number
  icons_simple?: number
  icons_text?: number
  icons_favicon?: number
  icons_dropped?: number
  warnings?: string[]
}

export interface ImportResult {
  ok: boolean
  categories_imported: number
  bookmarks_imported: number
  skipped: number
}

export interface UpdateInfo {
  current_version: string
  latest_version: string
  update_available: boolean
  release_url?: string
  checked_at?: string
}

export interface ExportBundle {
  format: string
  schema_version: number
  exported_at?: string
  checksum?: string
  categories?: Category[]
  bookmarks?: Bookmark[]
  settings?: Record<string, unknown>
}
