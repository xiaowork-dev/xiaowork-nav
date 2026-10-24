<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import AppHeader from '@/components/AppHeader.vue'
import SearchBar from '@/components/SearchBar.vue'
import CategorySection from '@/components/CategorySection.vue'
import BookmarkCard from '@/components/BookmarkCard.vue'
import { useSiteStore } from '@/stores/site'
import { searchBookmarks } from '@/api/public'
import type { Bookmark, SearchEngine } from '@/types/api'

const site = useSiteStore()
const query = ref('')
const results = ref<Bookmark[]>([])
const searching = ref(false)
const now = ref(new Date())

let searchTimer: number | undefined
let clockTimer: number | undefined

const engines = computed<SearchEngine[]>(() => site.settings?.search_engines ?? [])
const defaultEngine = computed(() => site.settings?.default_search_engine ?? '')
const defaultNewTab = computed(() => site.settings?.open_in_new_tab_default ?? true)

const clock = computed(() => {
  const h = String(now.value.getHours()).padStart(2, '0')
  const m = String(now.value.getMinutes()).padStart(2, '0')
  const s = String(now.value.getSeconds()).padStart(2, '0')
  return `${h}:${m}:${s}`
})

const dateLabel = computed(() =>
  now.value.toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric', weekday: 'long' }),
)

const showResults = computed(() => query.value.trim().length > 0)

watch(query, (value) => {
  window.clearTimeout(searchTimer)
  const q = value.trim()
  if (!q) {
    results.value = []
    searching.value = false
    return
  }
  searching.value = true
  searchTimer = window.setTimeout(async () => {
    try {
      results.value = (await searchBookmarks(q)).items
    } catch {
      // Fall back to a local filter when the API is unavailable.
      const lowered = q.toLowerCase()
      results.value = site.categories
        .flatMap((c) => c.bookmarks ?? [])
        .filter((b) =>
          [b.title, b.description, b.keywords].some((field) => field.toLowerCase().includes(lowered)),
        )
    } finally {
      searching.value = false
    }
  }, 250)
})

function openEngine(engine: SearchEngine): void {
  const q = query.value.trim()
  if (!q) return
  const url = engine.url_template.replace('{query}', encodeURIComponent(q))
  window.open(url, '_blank', 'noopener')
}

onMounted(() => {
  void site.load()
  clockTimer = window.setInterval(() => {
    now.value = new Date()
  }, 1000)
})

onUnmounted(() => {
  window.clearInterval(clockTimer)
  window.clearTimeout(searchTimer)
})
</script>

<template>
  <div class="page">
    <AppHeader />

    <section class="clock">
      <div class="clock__time">{{ clock }}</div>
      <div class="clock__date">{{ dateLabel }}</div>
    </section>

    <SearchBar
      v-model="query"
      :engines="engines"
      :default-engine="defaultEngine"
      @submit="openEngine"
    />

    <div v-if="site.loading" class="empty">
      <span class="spinner" style="margin: 0 auto 10px" />
      正在加载导航数据…
    </div>

    <div v-else-if="site.error" class="empty">
      <p>无法加载导航数据：{{ site.error }}</p>
      <p class="dim">
        若管理员已关闭游客访问，请先
        <RouterLink class="badge" to="/login">登录</RouterLink>
      </p>
    </div>

    <template v-else-if="showResults">
      <section class="category">
        <div class="category__head">
          <h2 class="category__title">搜索结果</h2>
          <span class="badge">{{ results.length }}</span>
          <span v-if="searching" class="spinner" />
        </div>
        <div v-if="results.length" class="tiles">
          <BookmarkCard
            v-for="bookmark in results"
            :key="bookmark.id"
            :bookmark="bookmark"
            :default-new-tab="defaultNewTab"
          />
        </div>
        <p v-else-if="!searching" class="dim" style="font-size: 0.86rem">没有匹配的站点</p>
      </section>
    </template>

    <template v-else>
      <CategorySection
        v-for="category in site.categories"
        :key="category.id"
        :category="category"
        :default-new-tab="defaultNewTab"
      />
      <div v-if="!site.categories.length" class="empty">还没有任何公开分类</div>
    </template>
  </div>
</template>
