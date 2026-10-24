/** Admin bookmark endpoints. */
import { api } from './client'
import type { Bookmark, BookmarkInput, OrderItem, OrderResult, Paginated } from '@/types/api'

export interface BookmarkQuery {
  category_id?: number
  q?: string
  is_visible?: boolean
  page?: number
  page_size?: number
}

export function listBookmarks(query: BookmarkQuery = {}): Promise<Paginated<Bookmark>> {
  return api.get<Paginated<Bookmark>>('/api/v1/admin/bookmarks', {
    query: {
      category_id: query.category_id,
      q: query.q,
      is_visible: query.is_visible,
      page: query.page,
      page_size: query.page_size,
    },
  })
}

export function getBookmark(id: number): Promise<Bookmark> {
  return api.get<Bookmark>(`/api/v1/admin/bookmarks/${id}`)
}

export function createBookmark(input: BookmarkInput): Promise<Bookmark> {
  return api.post<Bookmark>('/api/v1/admin/bookmarks', input)
}

export function updateBookmark(id: number, patch: Partial<BookmarkInput>): Promise<Bookmark> {
  return api.patch<Bookmark>(`/api/v1/admin/bookmarks/${id}`, patch)
}

export function deleteBookmark(id: number): Promise<void> {
  return api.delete<void>(`/api/v1/admin/bookmarks/${id}`)
}

export function reorderBookmarks(items: OrderItem[]): Promise<OrderResult> {
  return api.put<OrderResult>('/api/v1/admin/bookmarks/order', { items })
}

export function moveBookmark(id: number, categoryId: number, sortOrder?: number): Promise<Bookmark> {
  return api.post<Bookmark>(`/api/v1/admin/bookmarks/${id}/move`, {
    category_id: categoryId,
    sort_order: sortOrder,
  })
}
