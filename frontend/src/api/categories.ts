/** Admin category endpoints. */
import { api } from './client'
import type { Category, CategoryInput, OrderItem, OrderResult, Paginated } from '@/types/api'

export function listCategories(): Promise<Paginated<Category>> {
  return api.get<Paginated<Category>>('/api/v1/admin/categories')
}

export function getCategory(id: number): Promise<Category> {
  return api.get<Category>(`/api/v1/admin/categories/${id}`)
}

export function createCategory(input: CategoryInput): Promise<Category> {
  return api.post<Category>('/api/v1/admin/categories', input)
}

export function updateCategory(id: number, patch: Partial<CategoryInput>): Promise<Category> {
  return api.patch<Category>(`/api/v1/admin/categories/${id}`, patch)
}

export function deleteCategory(id: number): Promise<void> {
  return api.delete<void>(`/api/v1/admin/categories/${id}`)
}

export function reorderCategories(items: OrderItem[]): Promise<OrderResult> {
  return api.put<OrderResult>('/api/v1/admin/categories/order', { items })
}
