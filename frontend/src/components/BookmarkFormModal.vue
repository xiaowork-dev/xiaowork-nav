<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import ModalDialog from './ModalDialog.vue'
import * as bookmarksApi from '@/api/bookmarks'
import * as adminApi from '@/api/admin'
import { errorMessage, useToast } from '@/composables/useToast'
import type { Bookmark, Category, IconType } from '@/types/api'

const props = defineProps<{
  bookmark?: Bookmark | null
  categories: Category[]
  defaultCategoryId?: number
}>()

const emit = defineEmits<{ close: []; saved: [] }>()

const toast = useToast()
const busy = ref(false)
const error = ref<string | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)

const form = reactive({
  title: props.bookmark?.title ?? '',
  url: props.bookmark?.url ?? '',
  category_id: props.bookmark?.category_id ?? props.defaultCategoryId ?? props.categories[0]?.id ?? 0,
  description: props.bookmark?.description ?? '',
  keywords: props.bookmark?.keywords ?? '',
  icon_type: (props.bookmark?.icon_type ?? 'none') as IconType,
  icon_value: props.bookmark?.icon_value ?? '',
  sort_order: props.bookmark?.sort_order ?? 0,
  is_visible: props.bookmark?.is_visible ?? true,
  open_in_new_tab: props.bookmark?.open_in_new_tab ?? true,
})

const hasCategories = computed(() => props.categories.length > 0)

// Keep icon_type consistent when the address is typed manually: a non-empty
// value implies a URL icon, an empty value means no icon.
watch(
  () => form.icon_value,
  (value) => {
    const trimmed = value.trim()
    if (!trimmed) {
      form.icon_type = 'none'
    } else if (form.icon_type === 'none') {
      form.icon_type = 'url'
    }
  },
)

async function uploadIcon(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  busy.value = true
  error.value = null
  try {
    const result = await adminApi.uploadIcon(file)
    form.icon_type = 'upload'
    form.icon_value = result.url
    toast.success('图标已上传')
  } catch (e) {
    error.value = errorMessage(e)
  } finally {
    busy.value = false
    input.value = ''
  }
}

async function fetchIcon(): Promise<void> {
  if (!form.url.trim()) {
    error.value = '请先填写网站地址，再抓取图标'
    return
  }
  busy.value = true
  error.value = null
  try {
    const result = await adminApi.fetchIcon(form.url.trim())
    form.icon_type = result.icon_type
    form.icon_value = result.icon_value
    toast.success('图标已抓取')
  } catch (e) {
    error.value = errorMessage(e)
  } finally {
    busy.value = false
  }
}

function clearIcon(): void {
  form.icon_type = 'none'
  form.icon_value = ''
}

async function submit(): Promise<void> {
  if (!form.title.trim()) {
    error.value = '标题不能为空'
    return
  }
  if (!form.url.trim()) {
    error.value = '网站地址不能为空'
    return
  }
  if (!form.category_id) {
    error.value = '请选择所属分类'
    return
  }
  busy.value = true
  error.value = null
  try {
    const payload = {
      category_id: form.category_id,
      title: form.title.trim(),
      url: form.url.trim(),
      description: form.description,
      keywords: form.keywords,
      icon_type: form.icon_type,
      icon_value: form.icon_value,
      sort_order: form.sort_order,
      is_visible: form.is_visible,
      open_in_new_tab: form.open_in_new_tab,
    }
    if (props.bookmark) {
      await bookmarksApi.updateBookmark(props.bookmark.id, payload)
      toast.success('站点已更新')
    } else {
      await bookmarksApi.createBookmark(payload)
      toast.success('站点已创建')
    }
    emit('saved')
    emit('close')
  } catch (e) {
    error.value = errorMessage(e)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <ModalDialog :title="props.bookmark ? '编辑站点' : '新建站点'" @close="emit('close')">
    <form @submit.prevent="submit">
      <div class="field">
        <label for="bm-title">名称</label>
        <input id="bm-title" v-model="form.title" class="input" maxlength="128" required />
      </div>

      <div class="field">
        <label for="bm-url">网址</label>
        <input id="bm-url" v-model="form.url" class="input" placeholder="https://example.com" required />
      </div>

      <div class="field">
        <label for="bm-cat">所属分类</label>
        <select id="bm-cat" v-model.number="form.category_id" class="select" :disabled="!hasCategories" required>
          <option v-for="category in categories" :key="category.id" :value="category.id">
            {{ category.name }}
          </option>
        </select>
        <span v-if="!hasCategories" class="hint">请先在“分类管理”中创建分类</span>
      </div>

      <div class="field">
        <label>图标</label>
        <div class="row" style="flex-wrap: wrap">
          <input ref="fileInput" type="file" accept="image/png,image/jpeg,image/webp,image/gif,image/x-icon,image/svg+xml" hidden @change="uploadIcon" />
          <button type="button" class="btn btn--sm" :disabled="busy" @click="fileInput?.click()">上传图标</button>
          <button type="button" class="btn btn--sm" :disabled="busy" @click="fetchIcon">抓取站点图标</button>
          <button type="button" class="btn btn--sm btn--ghost" :disabled="busy" @click="clearIcon">清除</button>
        </div>
        <input v-model="form.icon_value" class="input" style="margin-top: 8px" placeholder="图标地址（可手动填写）" />
        <span class="hint">当前类型：{{ form.icon_type }}</span>
      </div>

      <div class="field">
        <label for="bm-desc">描述（可选）</label>
        <input id="bm-desc" v-model="form.description" class="input" maxlength="512" />
      </div>

      <div class="field">
        <label for="bm-keywords">关键词（可选，逗号分隔）</label>
        <input id="bm-keywords" v-model="form.keywords" class="input" maxlength="512" />
      </div>

      <div class="field">
        <label for="bm-order">排序值（越小越靠前）</label>
        <input id="bm-order" v-model.number="form.sort_order" class="input" type="number" />
      </div>

      <label class="checkbox" style="margin-bottom: 8px">
        <input v-model="form.is_visible" type="checkbox" />
        公开显示（关闭后仅管理员可见）
      </label>
      <br />
      <label class="checkbox">
        <input v-model="form.open_in_new_tab" type="checkbox" />
        在新标签页打开
      </label>

      <p v-if="error" class="error-text">{{ error }}</p>

      <div class="modal__actions">
        <button type="button" class="btn" :disabled="busy" @click="emit('close')">取消</button>
        <button type="submit" class="btn btn--primary" :disabled="busy || !hasCategories">
          {{ busy ? '保存中…' : '保存' }}
        </button>
      </div>
    </form>
  </ModalDialog>
</template>
