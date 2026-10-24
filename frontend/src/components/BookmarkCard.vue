<script setup lang="ts">
import { computed } from 'vue'
import type { Bookmark } from '@/types/api'
import IconImage from './IconImage.vue'

const props = defineProps<{ bookmark: Bookmark; defaultNewTab: boolean }>()

const target = computed(() => (props.bookmark.open_in_new_tab || props.defaultNewTab ? '_blank' : '_self'))
const tooltip = computed(() => {
  const parts = [props.bookmark.description, props.bookmark.tags].filter((v) => v && v.trim())
  return parts.join(' · ') || props.bookmark.title
})
</script>

<template>
  <a
    class="tile"
    :href="bookmark.url"
    :target="target"
    :rel="target === '_blank' ? 'noopener noreferrer' : undefined"
    :title="tooltip"
  >
    <span class="tile__icon">
      <IconImage
        :type="bookmark.icon_type"
        :value="bookmark.icon_value"
        :title="bookmark.title"
        :site-url="bookmark.url"
      />
    </span>
    <span class="tile__label" :style="bookmark.color ? { color: bookmark.color } : undefined">
      {{ bookmark.title }}
    </span>
  </a>
</template>
