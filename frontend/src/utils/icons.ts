/** Shared labels for bookmark icons, used by the admin UI. */
import type { IconSource, IconType } from '@/types/api'

export const ICON_TYPE_LABELS: Record<IconType, string> = {
  none: '无图标',
  url: '外部地址',
  upload: '本地上传',
  auto: '自动获取',
  favicon: '站点图标',
  simple: 'Simple Icons',
  text: '文字图标',
}

/**
 * How an automatically fetched icon was obtained. `brand-guess` is explicitly
 * labelled as a guess: the source cannot be verified as the official logo.
 */
export const ICON_SOURCE_LABELS: Record<IconSource, string> = {
  manual: '手动设置',
  'link-icon': '网页声明图标',
  'apple-touch': 'Apple Touch 图标',
  manifest: '应用清单图标',
  'favicon-ico': '站点 favicon.ico',
  'brand-guess': '品牌图标（推测）',
}

/** Icon types the automatic fetcher must never overwrite. */
export const MANUAL_ICON_TYPES: IconType[] = ['url', 'upload', 'simple', 'text']

export function isManualIconType(type: IconType): boolean {
  return MANUAL_ICON_TYPES.includes(type)
}

export function iconTypeLabel(type: IconType): string {
  return ICON_TYPE_LABELS[type] ?? type
}

export function iconSourceLabel(source?: IconSource): string {
  if (!source) return ''
  return ICON_SOURCE_LABELS[source] ?? source
}
