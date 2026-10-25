<script setup lang="ts">
import BookmarkGrid from './BookmarkGrid.vue'
import type { Category } from '@/types/api'

defineProps<{ category: Category; defaultNewTab: boolean }>()
</script>

<template>
  <!-- The id is the scroll anchor used by the sidebar and the mobile chip bar. -->
  <section :id="`category-${category.id}`" class="category" :data-category-id="category.id">
    <div class="category__head">
      <span v-if="category.icon" class="category__icon">{{ category.icon }}</span>
      <h2 class="category__title" :style="category.color ? { color: category.color } : undefined">
        {{ category.name }}
      </h2>
      <span class="badge">{{ category.bookmarks?.length ?? 0 }}</span>
      <span v-if="category.description" class="dim" style="font-size: 0.82rem">{{ category.description }}</span>
    </div>
    <BookmarkGrid v-if="category.bookmarks && category.bookmarks.length" :bookmarks="category.bookmarks" :default-new-tab="defaultNewTab" />
    <p v-else class="dim" style="font-size: 0.84rem">该分类暂无站点</p>
  </section>
</template>
