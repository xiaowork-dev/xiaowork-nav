<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import type { Category } from '@/types/api'

/**
 * Fixed category rail for the scroll layout (PC).
 *
 * It only mirrors `site.categories` — no data of its own — and reports clicks
 * upwards so the layout owns the scrolling behaviour.
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
  <aside class="sidebar" :class="{ 'sidebar--collapsed': collapsed }" aria-label="分类导航">
    <div class="sidebar__head">
      <span v-if="!collapsed" class="sidebar__title">分类</span>
      <button
        type="button"
        class="btn btn--sm btn--ghost sidebar__toggle"
        :title="collapsed ? '展开分类栏' : '折叠分类栏'"
        :aria-expanded="!collapsed"
        @click="emit('toggle')"
      >
        {{ collapsed ? '»' : '«' }}
      </button>
    </div>

    <nav ref="listEl" class="sidebar__list">
      <button
        v-for="category in categories"
        :key="category.id"
        type="button"
        class="sidebar__item"
        :class="{ 'is-active': category.id === activeId }"
        :data-id="category.id"
        :title="category.name"
        :aria-current="category.id === activeId ? 'true' : undefined"
        @click="emit('select', category.id)"
      >
        <span class="sidebar__icon" :style="category.color ? { color: category.color } : undefined">
          {{ category.icon || category.name.slice(0, 1) }}
        </span>
        <span v-if="!collapsed" class="sidebar__label">{{ category.name }}</span>
        <span v-if="!collapsed" class="sidebar__count">{{ count(category) }}</span>
      </button>
    </nav>
  </aside>
</template>
