<script setup lang="ts">
import type { Bookmark } from '@/types/api'
import IconImage from './IconImage.vue'

const props = defineProps<{ bookmark: Bookmark; defaultNewTab: boolean }>()

function target(): string {
  return props.bookmark.open_in_new_tab || props.defaultNewTab ? '_blank' : '_self'
}
</script>

<template>
  <a
    class="tile"
    :href="bookmark.url"
    :target="target()"
    :rel="target() === '_blank' ? 'noopener noreferrer' : undefined"
    :title="bookmark.description || bookmark.title"
  >
    <span class="tile__icon">
      <IconImage :type="bookmark.icon_type" :value="bookmark.icon_value" :title="bookmark.title" />
    </span>
    <span class="tile__label">{{ bookmark.title }}</span>
  </a>
</template>
