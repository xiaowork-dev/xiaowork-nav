/** Public (anonymous) read endpoints. */
import { api } from './client'
import type { Bookmark, Category, NavigationResponse, Paginated, PublicSettings } from '@/types/api'

export function getNavigation(): Promise<NavigationResponse> {
  return api.get<NavigationResponse>('/api/v1/public/navigation')
}

export function getPublicSettings(): Promise<PublicSettings> {
  return api.get<PublicSettings>('/api/v1/public/settings')
}

export function getPublicCategories(): Promise<Paginated<Category>> {
  return api.get<Paginated<Category>>('/api/v1/public/categories')
}

export function getCategoryBookmarks(categoryId: number): Promise<Paginated<Bookmark>> {
  return api.get<Paginated<Bookmark>>(`/api/v1/public/categories/${categoryId}/bookmarks`)
}

export function searchBookmarks(query: string, page = 1, pageSize = 20): Promise<Paginated<Bookmark>> {
  return api.get<Paginated<Bookmark>>('/api/v1/public/search', {
    query: { q: query, page, page_size: pageSize },
  })
}
