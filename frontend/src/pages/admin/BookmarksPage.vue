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

const categories = ref<Category[]>([])
const bookmarks = ref<Bookmark[]>([])
const loading = ref(false)
const filterCategory = ref(0)
const search = ref('')
const showForm = ref(false)
const editing = ref<Bookmark | null>(null)
const pendingDelete = ref<Bookmark | null>(null)
const deleting = ref(false)

const canReorder = computed(() => filterCategory.value !== 0 && search.value.trim() === '')

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
    const collected: Bookmark[] = []
    let page = 1
    // The API caps page_size at 100; walk pages to show everything.
    for (; page <= 20; page += 1) {
      const result = await bookmarksApi.listBookmarks({
        category_id: filterCategory.value || undefined,
        q: search.value.trim() || undefined,
        page,
        page_size: 100,
      })
      collected.push(...result.items)
      if (collected.length >= result.total || result.items.length === 0) break
    }
    bookmarks.value = collected
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
    void loadBookmarks()
  }, 300)
})

watch(filterCategory, () => {
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

    <div class="row" style="flex-wrap: wrap; margin-bottom: 12px">
      <select v-model.number="filterCategory" class="select" style="max-width: 220px">
        <option :value="0">全部分类</option>
        <option v-for="category in categories" :key="category.id" :value="category.id">
          {{ category.name }}{{ category.is_visible ? '' : '（私有）' }}
        </option>
      </select>
      <input v-model="search" class="input" style="max-width: 260px" placeholder="搜索名称/描述/关键词" />
      <span class="badge">{{ bookmarks.length }}</span>
    </div>

    <p class="hint" style="margin-top: 0">
      <template v-if="canReorder">拖拽调整顺序（手机端请使用 ↑ ↓ 按钮）。</template>
      <template v-else>选择具体分类且清空搜索后可拖拽排序；跨分类移动请在“编辑”中修改所属分类。</template>
    </p>

    <div v-if="loading" class="empty">加载中…</div>
    <div v-else-if="!bookmarks.length" class="empty">没有符合条件的站点。</div>

    <div v-else class="list">
      <div
        v-for="(bookmark, index) in bookmarks"
        :key="bookmark.id"
        class="list-item"
        :class="{ dragging: draggingId === bookmark.id, 'drag-over': overId === bookmark.id }"
        :draggable="canReorder"
        @dragstart="canReorder && onDragStart(bookmark)"
        @dragover="canReorder && onDragOver(bookmark, $event)"
        @drop="canReorder && onDrop(bookmark)"
        @dragend="onDragEnd"
      >
        <span v-if="canReorder" class="list-item__grip" title="拖拽排序">⠿</span>

        <span class="tile__icon" style="width: 36px; height: 36px; border-radius: 10px">
          <IconImage :type="bookmark.icon_type" :value="bookmark.icon_value" :title="bookmark.title" />
        </span>

        <div class="list-item__main">
          <div class="list-item__title">
            {{ bookmark.title }}
            <span v-if="!bookmark.is_visible" class="badge badge--private">私有</span>
          </div>
          <div class="list-item__meta">{{ bookmark.url }}</div>
        </div>

        <div v-if="canReorder" class="order-buttons">
          <button type="button" class="btn btn--sm btn--ghost" :disabled="index === 0" @click="move(index, -1)">↑</button>
          <button type="button" class="btn btn--sm btn--ghost" :disabled="index === bookmarks.length - 1" @click="move(index, 1)">↓</button>
        </div>

        <div class="list-item__actions">
          <button type="button" class="btn btn--sm" @click="toggleVisible(bookmark)">
            {{ bookmark.is_visible ? '设为私有' : '设为公开' }}
          </button>
          <button type="button" class="btn btn--sm" @click="openEdit(bookmark)">编辑</button>
          <button type="button" class="btn btn--sm btn--danger" @click="pendingDelete = bookmark">删除</button>
        </div>
      </div>
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
