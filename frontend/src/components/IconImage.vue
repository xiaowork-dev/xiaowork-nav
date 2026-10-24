<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { IconType } from '@/types/api'

/**
 * Icon display chain (all automatic steps are display-only fallbacks; the
 * stored icon value is never modified):
 *
 *   1. the icon the user set (url / upload / auto / simple / text)
 *   2. the site's own favicon (`<origin>/favicon.ico`)
 *   3. the first letter of the title
 */
const props = defineProps<{
  type: IconType
  value: string
  title: string
  /** Bookmark URL, used to derive the favicon fallback. */
  siteUrl?: string
}>()

const index = ref(0)

const faviconURL = computed(() => {
  const raw = (props.siteUrl ?? '').trim()
  if (!raw) return ''
  try {
    const url = new URL(raw)
    if (url.protocol !== 'http:' && url.protocol !== 'https:') return ''
    return `${url.origin}/favicon.ico`
  } catch {
    return ''
  }
})

const primaryURL = computed(() => {
  const value = (props.value ?? '').trim()
  switch (props.type) {
    case 'url':
    case 'upload':
    case 'auto':
      return value
    case 'simple':
      return value ? `https://cdn.simpleicons.org/${encodeURIComponent(value)}` : ''
    case 'favicon':
      return faviconURL.value
    default:
      return ''
  }
})

const isText = computed(() => props.type === 'text' && (props.value ?? '').trim() !== '')
const textValue = computed(() => (props.value ?? '').trim().slice(0, 4))

const candidates = computed<string[]>(() => {
  if (isText.value) return []
  const list: string[] = []
  const primary = primaryURL.value
  if (primary) list.push(primary)
  const favicon = faviconURL.value
  if (favicon && favicon !== primary) list.push(favicon)
  return list
})

const currentSrc = computed(() => candidates.value[index.value] ?? '')
const showLetter = computed(() => !isText.value && index.value >= candidates.value.length)
const letter = computed(() => (props.title.trim().charAt(0) || '?').toUpperCase())

watch(
  () => [props.type, props.value, props.siteUrl],
  () => {
    index.value = 0
  },
)

function onError(): void {
  index.value += 1
}
</script>

<template>
  <span v-if="isText" class="tile__text">{{ textValue }}</span>
  <img
    v-else-if="currentSrc"
    :src="currentSrc"
    :alt="title"
    loading="lazy"
    decoding="async"
    referrerpolicy="no-referrer"
    @error="onError"
  />
  <span v-else-if="showLetter" class="tile__fallback">{{ letter }}</span>
</template>
