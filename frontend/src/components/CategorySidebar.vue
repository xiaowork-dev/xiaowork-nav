<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import type { Category } from '@/types/api'

/**
 * Category rail for the scroll layout.
 *
 * It is a real column of the page shell: it sits on the page's left edge,
 * spans from below the header to the bottom of the viewport, and only ever
 * mirrors `site.categories` — the layout owns the scrolling behaviour.
 */
const props = defineProps<{
  categories: Category[]
  activeId: number | null
  collapsed: boolean
}>()

const emit = defineEmits<{
  (e: 'select', id: number): void
  (e: 'toggle'): void
}>()

const listEl = ref<HTMLElement | null>(null)

/** Keeps the highlighted entry inside the rail's own scroll viewport. */
watch(
  () => props.activeId,
  async (id) => {
    if (id === null || props.collapsed) return
    await nextTick()
    const target = listEl.value?.querySelector<HTMLElement>(`[data-id="${id}"]`)
    target?.scrollIntoView({ block: 'nearest' })
  },
)

function count(category: Category): number {
  return category.bookmarks?.length ?? 0
}
</script>

<template>
  <aside class="rail" :class="{ 'rail--collapsed': collapsed }" aria-label="分类导航">
    <div class="rail__head">
      <span v-if="!collapsed" class="rail__title">分类</span>
      <button
        type="button"
        class="rail__toggle"
        :title="collapsed ? '展开分类栏' : '折叠分类栏'"
        :aria-expanded="!collapsed"
        @click="emit('toggle')"
      >
        <span aria-hidden="true">{{ collapsed ? '»' : '«' }}</span>
        <span class="sr-only">{{ collapsed ? '展开分类栏' : '折叠分类栏' }}</span>
      </button>
    </div>

    <nav ref="listEl" class="rail__list">
      <button
        v-for="category in categories"
        :key="category.id"
        type="button"
        class="rail__item"
        :class="{ 'is-active': category.id === activeId }"
        :data-id="category.id"
        :title="category.name"
        :aria-current="category.id === activeId ? 'true' : undefined"
        @click="emit('select', category.id)"
      >
        <span class="rail__icon" :style="category.color ? { color: category.color } : undefined">
          {{ category.icon || category.name.slice(0, 1) }}
        </span>
        <template v-if="!collapsed">
          <span class="rail__label">{{ category.name }}</span>
          <span class="rail__count">{{ count(category) }}</span>
        </template>
      </button>
    </nav>
  </aside>
</template>
