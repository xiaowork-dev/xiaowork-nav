<script setup lang="ts">
import { onMounted, ref } from 'vue'
import * as categoriesApi from '@/api/categories'
import CategoryFormModal from '@/components/CategoryFormModal.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import { useDragSort } from '@/composables/useDragSort'
import { errorMessage, useToast } from '@/composables/useToast'
import type { Category, OrderItem } from '@/types/api'

const toast = useToast()

const categories = ref<Category[]>([])
const loading = ref(false)
const showForm = ref(false)
const editing = ref<Category | null>(null)
const pendingDelete = ref<Category | null>(null)
const deleting = ref(false)

async function load(): Promise<void> {
  loading.value = true
  try {
    categories.value = (await categoriesApi.listCategories()).items
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    loading.value = false
  }
}

async function commitOrder(ordered: Category[]): Promise<void> {
  const items: OrderItem[] = ordered.map((c, index) => ({ id: c.id, sort_order: (index + 1) * 10 }))
  try {
    await categoriesApi.reorderCategories(items)
    toast.success('排序已保存')
  } catch (e) {
    toast.error(errorMessage(e))
    await load()
  }
}

const { draggingId, overId, onDragStart, onDragEnd, onDragOver, onDrop } = useDragSort(
  categories,
  (c) => c.id,
  commitOrder,
)

async function move(index: number, delta: number): Promise<void> {
  const target = index + delta
  if (target < 0 || target >= categories.value.length) return
  const list = [...categories.value]
  const [item] = list.splice(index, 1)
  list.splice(target, 0, item)
  categories.value = list
  await commitOrder(list)
}

function openCreate(): void {
  editing.value = null
  showForm.value = true
}

function openEdit(category: Category): void {
  editing.value = category
  showForm.value = true
}

async function toggleVisible(category: Category): Promise<void> {
  try {
    await categoriesApi.updateCategory(category.id, { is_visible: !category.is_visible })
    await load()
  } catch (e) {
    toast.error(errorMessage(e))
  }
}

async function confirmDelete(): Promise<void> {
  if (!pendingDelete.value) return
  deleting.value = true
  try {
    await categoriesApi.deleteCategory(pendingDelete.value.id)
    toast.success('分类已删除')
    pendingDelete.value = null
    await load()
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    deleting.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="card">
    <div class="card__head">
      <h2 class="card__title">分类管理</h2>
      <button type="button" class="btn btn--primary btn--sm" @click="openCreate">新建分类</button>
    </div>

    <p class="hint" style="margin-top: 0">
      拖拽可调整顺序（手机端请使用 ↑ ↓ 按钮）。删除分类会同时删除其下所有站点。
    </p>

    <div v-if="loading" class="empty">加载中…</div>

    <div v-else-if="!categories.length" class="empty">还没有分类，点击“新建分类”开始。</div>

    <div v-else class="list">
      <div
        v-for="(category, index) in categories"
        :key="category.id"
        class="list-item"
        :class="{ dragging: draggingId === category.id, 'drag-over': overId === category.id }"
        draggable="true"
        @dragstart="onDragStart(category)"
        @dragover="onDragOver(category, $event)"
        @drop="onDrop(category)"
        @dragend="onDragEnd"
      >
        <span class="list-item__grip" title="拖拽排序">⠿</span>

        <div class="list-item__main">
          <div class="list-item__title">
            {{ category.name }}
            <span v-if="!category.is_visible" class="badge badge--private">私有</span>
          </div>
          <div class="list-item__meta">
            {{ category.bookmark_count ?? 0 }} 个站点 · 排序值 {{ category.sort_order }}
            <template v-if="category.description"> · {{ category.description }}</template>
          </div>
        </div>

        <div class="order-buttons">
          <button type="button" class="btn btn--sm btn--ghost" :disabled="index === 0" @click="move(index, -1)">↑</button>
          <button type="button" class="btn btn--sm btn--ghost" :disabled="index === categories.length - 1" @click="move(index, 1)">↓</button>
        </div>

        <div class="list-item__actions">
          <button type="button" class="btn btn--sm" @click="toggleVisible(category)">
            {{ category.is_visible ? '设为私有' : '设为公开' }}
          </button>
          <button type="button" class="btn btn--sm" @click="openEdit(category)">编辑</button>
          <button type="button" class="btn btn--sm btn--danger" @click="pendingDelete = category">删除</button>
        </div>
      </div>
    </div>
  </div>

  <CategoryFormModal
    v-if="showForm"
    :category="editing"
    @close="showForm = false"
    @saved="load"
  />

  <ConfirmDialog
    v-if="pendingDelete"
    :message="`确定删除分类“${pendingDelete.name}”及其下所有站点吗？此操作不可撤销。`"
    :busy="deleting"
    @cancel="pendingDelete = null"
    @confirm="confirmDelete"
  />
</template>
