/**
 * API types mirroring `docs/openapi/openapi.yaml`.
 *
 * These are hand-maintained for now. Once the contract is frozen they can be
 * regenerated with `npm run gen:types` (openapi-typescript) into
 * `src/types/openapi.d.ts`; this file then becomes a thin re-export.
 */

export type Theme = 'light' | 'dark' | 'auto'
export type IconType = 'none' | 'url' | 'upload' | 'auto'
export type HomepageLayout = 'grid' | 'list'
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
  sort_order: number
  is_visible: boolean
  bookmark_count?: number
  created_at?: string
  updated_at?: string
  bookmarks?: Bookmark[]
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
  sort_order: number
  is_visible: boolean
  open_in_new_tab: boolean
  created_at?: string
  updated_at?: string
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

export interface OrderRequest {
  items: Array<{ id: number; sort_order: number }>
}

export interface OrderResult {
  ok: boolean
  updated?: number
}
