<script setup lang="ts">
import BookmarkCard from './BookmarkCard.vue'
import type { Bookmark } from '@/types/api'

/**
 * The tile grid shared by both navigation layouts and by search results, so
 * the two layouts cannot drift apart visually or in behaviour.
 */
defineProps<{ bookmarks: Bookmark[]; defaultNewTab: boolean }>()
</script>

<template>
  <div class="tiles">
    <BookmarkCard
      v-for="bookmark in bookmarks"
      :key="bookmark.id"
      :bookmark="bookmark"
      :default-new-tab="defaultNewTab"
    />
  </div>
</template>
