/** Admin settings, icons and data-management endpoints. */
import { api } from './client'
import type {
  AdminSettings,
  Backup,
  BulkIconRequest,
  ExportBundle,
  IconFillOutcome,
  IconJobSnapshot,
  IconRef,
  IconSummary,
  ImportPreview,
  ImportResult,
  UpdateInfo,
  UploadResult,
} from '@/types/api'

export function getAdminSettings(): Promise<AdminSettings> {
  return api.get<AdminSettings>('/api/v1/admin/settings')
}

export function updateAdminSettings(patch: Partial<AdminSettings>): Promise<AdminSettings> {
  return api.put<AdminSettings>('/api/v1/admin/settings', patch)
}

export function uploadIcon(file: File): Promise<UploadResult> {
  const form = new FormData()
  form.append('file', file)
  return api.post<UploadResult>('/api/v1/admin/uploads/icon', form, { raw: true })
}

export function fetchIcon(url: string): Promise<IconRef> {
  return api.post<IconRef>('/api/v1/admin/icons/fetch', { url })
}

/** Fetches and stores an icon for an existing bookmark. */
export function autoBookmarkIcon(bookmarkId: number, force = false): Promise<IconFillOutcome> {
  return api.post<IconFillOutcome>(`/api/v1/admin/bookmarks/${bookmarkId}/icon/auto`, { force })
}

export function getIconSummary(): Promise<IconSummary> {
  return api.get<IconSummary>('/api/v1/admin/icons/summary')
}

/** Starts the bulk icon completion job (only one may run at a time). */
export function startBulkIconFetch(payload: BulkIconRequest = {}): Promise<{ job: IconJobSnapshot }> {
  return api.post<{ job: IconJobSnapshot }>('/api/v1/admin/icons/bulk', payload)
}

export function getBulkIconStatus(): Promise<{ job: IconJobSnapshot | null }> {
  return api.get<{ job: IconJobSnapshot | null }>('/api/v1/admin/icons/bulk')
}

export function cancelBulkIconFetch(): Promise<{ canceled: boolean; job: IconJobSnapshot | null }> {
  return api.post<{ canceled: boolean; job: IconJobSnapshot | null }>('/api/v1/admin/icons/bulk/cancel')
}

export function exportData(): Promise<ExportBundle> {
  return api.get<ExportBundle>('/api/v1/admin/export')
}

export function previewImport(bundle: unknown): Promise<ImportPreview> {
  return api.post<ImportPreview>('/api/v1/admin/import/preview', bundle)
}

/**
 * Runs an import. The payload may be a native export object or a legacy
 * category array, so the conflict strategy is always sent as a query
 * parameter (it cannot be merged into an array body).
 */
export function runImport(bundle: unknown, strategy: string): Promise<ImportResult> {
  return api.post<ImportResult>('/api/v1/admin/import', bundle, {
    query: { conflict_strategy: strategy },
  })
}

export function listBackups(): Promise<{ items: Backup[] }> {
  return api.get<{ items: Backup[] }>('/api/v1/admin/backups')
}

export function createBackup(): Promise<Backup> {
  return api.post<Backup>('/api/v1/admin/backups')
}

export function restoreBackup(id: string): Promise<{ ok: boolean }> {
  return api.post<{ ok: boolean }>(`/api/v1/admin/backups/${encodeURIComponent(id)}/restore`)
}

export function checkUpdate(): Promise<UpdateInfo> {
  return api.get<UpdateInfo>('/api/v1/admin/update-check')
}
