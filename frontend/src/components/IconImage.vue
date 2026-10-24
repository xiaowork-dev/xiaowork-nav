<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { IconType } from '@/types/api'

/**
 * Icon display chain. Every automatic step is a display-only fallback; the
 * stored icon value is never modified.
 *
 *   1. the icon the user set (url / upload / simple / text)
 *   2. the site's own favicon (`<origin>/favicon.ico`)
 *   3. the first letter of the title
 *
 * The letter is always painted underneath and the image is revealed on top
 * once it has actually loaded, so a tile is never an empty box while an image
 * is still loading or hanging. The chain only advances on a real failure
 * (`error`) or on a degenerate image (a 1x1 tracking pixel), which keeps lazy
 * loading safe: a not-yet-requested image is never mistaken for a broken one.
 */
const props = defineProps<{
  type: IconType
  value: string
  title: string
  /** Bookmark URL, used to derive the favicon fallback. */
  siteUrl?: string
}>()

const index = ref(0)
const loaded = ref(false)

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
const exhausted = computed(() => !isText.value && index.value >= candidates.value.length)
const letter = computed(() => (props.title.trim().charAt(0) || '?').toUpperCase())

function advance(): void {
  loaded.value = false
  index.value += 1
}

function onLoad(event: Event): void {
  const img = event.target as HTMLImageElement
  // Ignore tracking pixels / transparent placeholders: covering the letter
  // with an effectively empty image would leave a blank tile.
  if (img.naturalWidth <= 1 || img.naturalHeight <= 1) {
    advance()
    return
  }
  loaded.value = true
}

watch(
  () => [props.type, props.value, props.siteUrl],
  () => {
    index.value = 0
    loaded.value = false
  },
)
</script>

<template>
  <span v-if="isText" class="tile__text">{{ textValue }}</span>
  <template v-else>
    <span class="tile__letter" :class="{ 'is-covered': loaded }" aria-hidden="true">{{ letter }}</span>
    <img
      v-if="currentSrc"
      class="tile__img"
      :class="{ 'is-loaded': loaded }"
      :src="currentSrc"
      :alt="title"
      loading="lazy"
      decoding="async"
      referrerpolicy="no-referrer"
      @load="onLoad"
      @error="advance"
    />
    <span v-if="exhausted" class="sr-only">{{ title }}（无可用图标）</span>
  </template>
</template>
