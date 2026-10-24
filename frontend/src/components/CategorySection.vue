<script setup lang="ts">
import type { Category } from '@/types/api'
import BookmarkCard from './BookmarkCard.vue'

defineProps<{ category: Category; defaultNewTab: boolean }>()
</script>

<template>
  <section class="category">
    <div class="category__head">
      <h2 class="category__title">{{ category.name }}</h2>
      <span class="badge">{{ category.bookmarks?.length ?? 0 }}</span>
      <span v-if="category.description" class="dim" style="font-size: 0.82rem">{{ category.description }}</span>
    </div>
    <div v-if="category.bookmarks && category.bookmarks.length" class="tiles">
      <BookmarkCard
        v-for="bookmark in category.bookmarks"
        :key="bookmark.id"
        :bookmark="bookmark"
        :default-new-tab="defaultNewTab"
      />
    </div>
    <p v-else class="dim" style="font-size: 0.84rem">该分类暂无站点</p>
  </section>
</template>
