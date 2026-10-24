<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import * as bookmarksApi from '@/api/bookmarks'
import * as categoriesApi from '@/api/categories'
import BookmarkFormModal from '@/components/BookmarkFormModal.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import IconImage from '@/components/IconImage.vue'
import { useDragSort } from '@/composables/useDragSort'
import { errorMessage, useToast } from '@/composables/useToast'
import type { Bookmark, Category, OrderItem } from '@/types/api'

const toast = useToast()

const PAGE_SIZES = [20, 50, 100]

const categories = ref<Category[]>([])
const bookmarks = ref<Bookmark[]>([])
const loading = ref(false)
const filterCategory = ref(0)
const search = ref('')
const page = ref(1)
const pageSize = ref(PAGE_SIZES[0])
const total = ref(0)

/**
 * Reordering only makes sense with an unambiguous list, so it is gated behind
 * an explicit sort mode: the selected category is loaded in full (no paging)
 * and drag/↑↓ reorder that complete list.
 */
const sortMode = ref(false)

const showForm = ref(false)
const editing = ref<Bookmark | null>(null)
const pendingDelete = ref<Bookmark | null>(null)
const deleting = ref(false)

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))
const canEnterSortMode = computed(() => filterCategory.value !== 0 && search.value.trim() === '')
const rangeStart = computed(() => (total.value === 0 ? 0 : (page.value - 1) * pageSize.value + 1))
const rangeEnd = computed(() => Math.min(page.value * pageSize.value, total.value))

async function loadCategories(): Promise<void> {
  try {
    categories.value = (await categoriesApi.listCategories()).items
  } catch (e) {
    toast.error(errorMessage(e))
  }
}

async function loadBookmarks(): Promise<void> {
  loading.value = true
  try {
    if (sortMode.value) {
      // Load every bookmark of the selected category so the order is complete.
      const collected: Bookmark[] = []
      for (let p = 1; p <= 20; p += 1) {
        const result = await bookmarksApi.listBookmarks({
          category_id: filterCategory.value || undefined,
          q: search.value.trim() || undefined,
          page: p,
          page_size: 100,
        })
        collected.push(...result.items)
        total.value = result.total
        if (collected.length >= result.total || result.items.length === 0) break
      }
      bookmarks.value = collected
      return
    }

    const result = await bookmarksApi.listBookmarks({
      category_id: filterCategory.value || undefined,
      q: search.value.trim() || undefined,
      page: page.value,
      page_size: pageSize.value,
    })
    bookmarks.value = result.items
    total.value = result.total

    // If the current page became empty (e.g. after a delete), step back.
    if (result.items.length === 0 && page.value > 1) {
      page.value = Math.max(1, Math.ceil(result.total / pageSize.value))
      await loadBookmarks()
    }
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    loading.value = false
  }
}

async function refresh(): Promise<void> {
  await Promise.all([loadCategories(), loadBookmarks()])
}

async function commitOrder(ordered: Bookmark[]): Promise<void> {
  const items: OrderItem[] = ordered.map((b, index) => ({ id: b.id, sort_order: (index + 1) * 10 }))
  try {
    await bookmarksApi.reorderBookmarks(items)
    toast.success('排序已保存')
  } catch (e) {
    toast.error(errorMessage(e))
    await loadBookmarks()
  }
}

const { draggingId, overId, onDragStart, onDragEnd, onDragOver, onDrop } = useDragSort(
  bookmarks,
  (b) => b.id,
  commitOrder,
)

async function move(index: number, delta: number): Promise<void> {
  const target = index + delta
  if (target < 0 || target >= bookmarks.value.length) return
  const list = [...bookmarks.value]
  const [item] = list.splice(index, 1)
  list.splice(target, 0, item)
  bookmarks.value = list
  await commitOrder(list)
}

async function toggleSortMode(): Promise<void> {
  if (!sortMode.value) {
    if (!canEnterSortMode.value) {
      toast.info('请先选择具体分类并清空搜索，再进入排序模式')
      return
    }
    sortMode.value = true
  } else {
    sortMode.value = false
  }
  page.value = 1
  await loadBookmarks()
}

function goToPage(target: number): void {
  const next = Math.min(Math.max(1, target), totalPages.value)
  if (next === page.value) return
  page.value = next
  void loadBookmarks()
}

function changePageSize(size: number): void {
  pageSize.value = size
  page.value = 1
  void loadBookmarks()
}

function openCreate(): void {
  editing.value = null
  showForm.value = true
}

function openEdit(bookmark: Bookmark): void {
  editing.value = bookmark
  showForm.value = true
}

async function toggleVisible(bookmark: Bookmark): Promise<void> {
  try {
    await bookmarksApi.updateBookmark(bookmark.id, { is_visible: !bookmark.is_visible })
    await loadBookmarks()
  } catch (e) {
    toast.error(errorMessage(e))
  }
}

async function confirmDelete(): Promise<void> {
  if (!pendingDelete.value) return
  deleting.value = true
  try {
    await bookmarksApi.deleteBookmark(pendingDelete.value.id)
    toast.success('站点已删除')
    pendingDelete.value = null
    await refresh()
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    deleting.value = false
  }
}

let searchTimer: number | undefined
watch(search, () => {
  window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(() => {
    page.value = 1
    void loadBookmarks()
  }, 300)
})

watch(filterCategory, () => {
  page.value = 1
  void loadBookmarks()
})

onMounted(refresh)
</script>

<template>
  <div class="card">
    <div class="card__head">
      <h2 class="card__title">网站管理</h2>
      <button type="button" class="btn btn--primary btn--sm" :disabled="!categories.length" @click="openCreate">
        新建站点
      </button>
    </div>

    <div class="toolbar">
      <select v-model.number="filterCategory" class="select" aria-label="按分类筛选">
        <option :value="0">全部分类</option>
        <option v-for="category in categories" :key="category.id" :value="category.id">
          {{ category.name }}{{ category.is_visible ? '' : '（私有）' }}
        </option>
      </select>
      <input v-model="search" class="input" placeholder="搜索名称/描述/关键词" aria-label="搜索站点" />
      <select
        v-if="!sortMode"
        :value="pageSize"
        class="select"
        aria-label="每页条数"
        @change="changePageSize(Number(($event.target as HTMLSelectElement).value))"
      >
        <option v-for="size in PAGE_SIZES" :key="size" :value="size">每页 {{ size }} 条</option>
      </select>
      <button
        type="button"
        class="btn btn--sm"
        :class="{ 'btn--primary': sortMode }"
        :title="canEnterSortMode ? '' : '请先选择具体分类并清空搜索'"
        @click="toggleSortMode"
      >
        {{ sortMode ? '退出排序模式' : '排序模式' }}
      </button>
    </div>

    <div v-if="sortMode" class="banner">
      <strong>排序模式</strong>
      <span>已加载「{{ categories.find((c) => c.id === filterCategory)?.name }}」的全部 {{ bookmarks.length }} 个站点，
        拖拽或使用 ↑ ↓ 调整顺序。</span>
    </div>

    <p v-if="!sortMode" class="hint" style="margin-top: 0">
      列表按页显示；需要调整站点顺序时，请选择具体分类并进入「排序模式」。
    </p>

    <div v-if="loading" class="empty">加载中…</div>
    <div v-else-if="!bookmarks.length" class="empty">没有符合条件的站点。</div>

    <div v-else class="list" :class="{ 'list--sortable': sortMode }">
      <div
        v-for="(bookmark, index) in bookmarks"
        :key="bookmark.id"
        class="list-item"
        :class="{ dragging: draggingId === bookmark.id, 'drag-over': overId === bookmark.id }"
        :draggable="sortMode"
        @dragstart="sortMode && onDragStart(bookmark)"
        @dragover="sortMode && onDragOver(bookmark, $event)"
        @drop="sortMode && onDrop(bookmark)"
        @dragend="onDragEnd"
      >
        <span class="list-item__grip" title="拖拽排序">⠿</span>

        <div class="list-item__main">
          <div class="list-item__title">
            {{ bookmark.title }}
            <span v-if="!bookmark.is_visible" class="badge badge--private">私有</span>
          </div>
          <div class="list-item__meta">
            {{ bookmark.url }}
            <template v-if="bookmark.tags"> · 标签：{{ bookmark.tags }}</template>
            · 图标：{{ bookmark.icon_type }}
          </div>
        </div>

        <div class="order-buttons">
          <button type="button" class="btn btn--sm btn--ghost" :disabled="index === 0" title="上移" @click="move(index, -1)">↑</button>
          <button type="button" class="btn btn--sm btn--ghost" :disabled="index === bookmarks.length - 1" title="下移" @click="move(index, 1)">↓</button>
        </div>

        <div class="list-item__actions">
          <span class="tile__icon" style="width: 36px; height: 36px; border-radius: 10px">
            <IconImage :type="bookmark.icon_type" :value="bookmark.icon_value" :title="bookmark.title" :site-url="bookmark.url" />
          </span>
          <button type="button" class="btn btn--sm" @click="toggleVisible(bookmark)">
            {{ bookmark.is_visible ? '设为私有' : '设为公开' }}
          </button>
          <button type="button" class="btn btn--sm" @click="openEdit(bookmark)">编辑</button>
          <button type="button" class="btn btn--sm btn--danger" @click="pendingDelete = bookmark">删除</button>
        </div>
      </div>
    </div>

    <div v-if="!sortMode && total > 0" class="pager">
      <span class="pager__info">
        第 {{ rangeStart }}–{{ rangeEnd }} 条，共 {{ total }} 条 · 第 {{ page }} / {{ totalPages }} 页
      </span>
      <span class="pager__controls">
        <button type="button" class="btn btn--sm" :disabled="page <= 1 || loading" @click="goToPage(1)">首页</button>
        <button type="button" class="btn btn--sm" :disabled="page <= 1 || loading" @click="goToPage(page - 1)">上一页</button>
        <button type="button" class="btn btn--sm" :disabled="page >= totalPages || loading" @click="goToPage(page + 1)">下一页</button>
        <button type="button" class="btn btn--sm" :disabled="page >= totalPages || loading" @click="goToPage(totalPages)">末页</button>
      </span>
    </div>
  </div>

  <BookmarkFormModal
    v-if="showForm"
    :bookmark="editing"
    :categories="categories"
    :default-category-id="filterCategory || undefined"
    @close="showForm = false"
    @saved="refresh"
  />

  <ConfirmDialog
    v-if="pendingDelete"
    :message="`确定删除站点“${pendingDelete.title}”吗？`"
    :busy="deleting"
    @cancel="pendingDelete = null"
    @confirm="confirmDelete"
  />
</template>
