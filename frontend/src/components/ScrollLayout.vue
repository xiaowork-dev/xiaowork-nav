<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import BookmarkGrid from './BookmarkGrid.vue'
import CategorySection from './CategorySection.vue'
import CategorySidebar from './CategorySidebar.vue'
import SearchBar from './SearchBar.vue'
import { useNavSearch } from '@/composables/useNavSearch'
import { useSiteStore } from '@/stores/site'
import { clampScrollTop, isAtBottom, pickActiveIndex } from '@/utils/scrollSpy'

/**
 * Scroll layout: every category stacked vertically, with a fixed category rail
 * on the left (PC) and a sticky category chip bar on narrow screens.
 *
 * Clicking a category smooth-scrolls to it; scrolling updates the highlight.
 * Both read the same `site.categories`, so there is only ever one data source.
 */
const site = useSiteStore()
const { query, results, searching, engines, defaultEngine, showResults, openEngine } = useNavSearch()

/** Distance from the viewport top that counts as "the section being read". */
const ACTIVE_OFFSET = 120
/** Gap left above a section when jumping to it. */
const SCROLL_GAP = 24

const SIDEBAR_KEY = 'xn-sidebar-collapsed'

const activeId = ref<number | null>(null)
const sidebarCollapsed = ref(localStorage.getItem(SIDEBAR_KEY) === '1')
const now = ref(new Date())
let clockTimer: number | undefined
let ticking = false

const clock = computed(() => {
  const h = String(now.value.getHours()).padStart(2, '0')
  const m = String(now.value.getMinutes()).padStart(2, '0')
  const s = String(now.value.getSeconds()).padStart(2, '0')
  return `${h}:${m}:${s}`
})

const dateLabel = computed(() =>
  now.value.toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric', weekday: 'long' }),
)

function sections(): HTMLElement[] {
  return Array.from(document.querySelectorAll<HTMLElement>('.category[data-category-id]'))
}

function idOf(el: HTMLElement): number | null {
  const raw = el.dataset.categoryId
  if (!raw) return null
  const parsed = Number(raw)
  return Number.isFinite(parsed) ? parsed : null
}

/**
 * Picks the category whose section currently owns the reading position.
 *
 * A plain scroll measurement is used rather than IntersectionObserver because
 * it also resolves the two cases observers handle poorly: several sections
 * inside the viewport at once, and the page being scrolled to the very bottom.
 */
function computeActive(): void {
  const list = sections()
  if (list.length === 0) {
    activeId.value = null
    return
  }

  const doc = document.documentElement
  const tops = list.map((el) => el.getBoundingClientRect().top)
  const atEnd = isAtBottom(window.scrollY, window.innerHeight, doc.scrollHeight)
  const index = pickActiveIndex(tops, ACTIVE_OFFSET, atEnd)
  activeId.value = index >= 0 ? idOf(list[index]) : null
}

function onScroll(): void {
  if (ticking) return
  ticking = true
  window.requestAnimationFrame(() => {
    ticking = false
    computeActive()
  })
}

function scrollToCategory(id: number): void {
  const el = document.getElementById(`category-${id}`)
  if (!el) return
  // Set the highlight immediately so the click feels instant.
  activeId.value = id
  const top = el.getBoundingClientRect().top + window.scrollY - SCROLL_GAP
  window.scrollTo({ top: clampScrollTop(top), behavior: 'smooth' })
}

function toggleSidebar(): void {
  sidebarCollapsed.value = !sidebarCollapsed.value
  localStorage.setItem(SIDEBAR_KEY, sidebarCollapsed.value ? '1' : '0')
}

watch(
  () => site.categories.length,
  () => {
    if (activeId.value === null) computeActive()
  },
)

onMounted(() => {
  clockTimer = window.setInterval(() => {
    now.value = new Date()
  }, 1000)
  window.addEventListener('scroll', onScroll, { passive: true })
  window.addEventListener('resize', onScroll)
  computeActive()
})

onUnmounted(() => {
  window.clearInterval(clockTimer)
  window.removeEventListener('scroll', onScroll)
  window.removeEventListener('resize', onScroll)
})
</script>

<template>
  <div class="scroll-layout">
    <section class="clock">
      <div class="clock__time">{{ clock }}</div>
      <div class="clock__date">{{ dateLabel }}</div>
    </section>

    <SearchBar v-model="query" :engines="engines" :default-engine="defaultEngine" @submit="openEngine" />

    <template v-if="showResults">
      <section class="category">
        <div class="category__head">
          <h2 class="category__title">搜索结果</h2>
          <span class="badge">{{ results.length }}</span>
          <span v-if="searching" class="spinner" />
        </div>
        <BookmarkGrid v-if="results.length" :bookmarks="results" :default-new-tab="site.settings?.open_in_new_tab_default ?? true" />
        <p v-else-if="!searching" class="dim" style="font-size: 0.86rem">没有匹配的站点</p>
      </section>
    </template>

    <div v-else-if="site.categories.length" class="scroll-layout__body" :class="{ 'is-collapsed': sidebarCollapsed }">
      <CategorySidebar
        class="scroll-layout__sidebar"
        :categories="site.categories"
        :active-id="activeId"
        :collapsed="sidebarCollapsed"
        @select="scrollToCategory"
        @toggle="toggleSidebar"
      />

      <div class="scroll-layout__content">
        <!-- Narrow screens: a sticky chip bar replaces the rail. -->
        <nav class="chipbar" aria-label="分类快速跳转">
          <button
            v-for="category in site.categories"
            :key="category.id"
            type="button"
            class="chipbar__item"
            :class="{ 'is-active': category.id === activeId }"
            @click="scrollToCategory(category.id)"
          >
            <span v-if="category.icon">{{ category.icon }}</span>
            {{ category.name }}
          </button>
        </nav>

        <CategorySection
          v-for="category in site.categories"
          :key="category.id"
          :category="category"
          :default-new-tab="site.settings?.open_in_new_tab_default ?? true"
        />
      </div>
    </div>

    <div v-else-if="!site.loading && !site.error" class="empty">还没有任何公开分类</div>
  </div>
</template>
