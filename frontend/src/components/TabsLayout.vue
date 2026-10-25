<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import BookmarkGrid from './BookmarkGrid.vue'
import SearchBar from './SearchBar.vue'
import { useNavSearch } from '@/composables/useNavSearch'
import { useSiteStore } from '@/stores/site'
import type { Category } from '@/types/api'

/**
 * Tabs layout: one category at a time behind a horizontal tab strip, with the
 * search box centred under the clock.
 *
 * The data is the same `site.categories` the scroll layout uses — only the way
 * it is presented differs, so a site can never appear in one layout and be
 * missing from the other.
 */
const site = useSiteStore()
const { query, results, searching, engines, defaultEngine, showResults, openEngine } = useNavSearch()

/** The pseudo-tab shown first: every site, in one grid. */
const HOME_TAB = 0

const activeTab = ref<number>(HOME_TAB)
const now = ref(new Date())
let clockTimer: number | undefined

const clock = computed(() => {
  const h = String(now.value.getHours()).padStart(2, '0')
  const m = String(now.value.getMinutes()).padStart(2, '0')
  const s = String(now.value.getSeconds()).padStart(2, '0')
  return `${h}:${m}:${s}`
})

const dateLabel = computed(() =>
  now.value.toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric', weekday: 'long' }),
)

/** Every visible bookmark, used by the 首页 tab (there is no favourites model yet). */
const allBookmarks = computed(() => site.categories.flatMap((c) => c.bookmarks ?? []))

const activeCategory = computed<Category | null>(
  () => site.categories.find((c) => c.id === activeTab.value) ?? null,
)

const visibleBookmarks = computed(() => {
  if (activeTab.value === HOME_TAB) return allBookmarks.value
  return activeCategory.value?.bookmarks ?? []
})

const defaultNewTab = computed(() => site.settings?.open_in_new_tab_default ?? true)

// Keep the selection valid if the category list changes (e.g. after a reload).
watch(
  () => site.categories.map((c) => c.id).join(','),
  () => {
    if (activeTab.value !== HOME_TAB && !activeCategory.value) activeTab.value = HOME_TAB
  },
)

onMounted(() => {
  clockTimer = window.setInterval(() => {
    now.value = new Date()
  }, 1000)
})

onUnmounted(() => window.clearInterval(clockTimer))
</script>

<template>
  <div class="tabs-layout">
    <section class="tabs-hero">
      <div class="clock">
        <div class="clock__time">{{ clock }}</div>
        <div class="clock__date">{{ dateLabel }}</div>
      </div>
      <SearchBar v-model="query" :engines="engines" :default-engine="defaultEngine" @submit="openEngine" />
    </section>

    <template v-if="showResults">
      <section class="category">
        <div class="category__head">
          <h2 class="category__title">搜索结果</h2>
          <span class="badge">{{ results.length }}</span>
          <span v-if="searching" class="spinner" />
        </div>
        <BookmarkGrid v-if="results.length" :bookmarks="results" :default-new-tab="defaultNewTab" />
        <p v-else-if="!searching" class="dim" style="font-size: 0.86rem">没有匹配的站点</p>
      </section>
    </template>

    <template v-else>
      <nav class="tabs" role="tablist" aria-label="分类标签">
        <button
          type="button"
          class="tabs__item"
          role="tab"
          :class="{ 'is-active': activeTab === HOME_TAB }"
          :aria-selected="activeTab === HOME_TAB"
          @click="activeTab = HOME_TAB"
        >
          首页
          <span class="tabs__count">{{ allBookmarks.length }}</span>
        </button>
        <button
          v-for="category in site.categories"
          :key="category.id"
          type="button"
          class="tabs__item"
          role="tab"
          :class="{ 'is-active': category.id === activeTab }"
          :aria-selected="category.id === activeTab"
          @click="activeTab = category.id"
        >
          <span v-if="category.icon" class="tabs__icon">{{ category.icon }}</span>
          {{ category.name }}
          <span class="tabs__count">{{ category.bookmarks?.length ?? 0 }}</span>
        </button>
      </nav>

      <section class="tabs-panel">
        <header class="tabs-panel__head">
          <h2 class="tabs-panel__title">
            {{ activeTab === HOME_TAB ? '全部站点' : activeCategory?.name }}
          </h2>
          <span class="badge">{{ visibleBookmarks.length }}</span>
          <span v-if="activeTab !== HOME_TAB && activeCategory?.description" class="dim" style="font-size: 0.82rem">
            {{ activeCategory.description }}
          </span>
        </header>

        <BookmarkGrid v-if="visibleBookmarks.length" :bookmarks="visibleBookmarks" :default-new-tab="defaultNewTab" />
        <p v-else class="dim" style="font-size: 0.86rem">该分类暂无站点</p>
      </section>
    </template>
  </div>
</template>
