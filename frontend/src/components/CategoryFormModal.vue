<script setup lang="ts">
import { reactive, ref } from 'vue'
import ModalDialog from './ModalDialog.vue'
import * as categoriesApi from '@/api/categories'
import { errorMessage, useToast } from '@/composables/useToast'
import type { Category } from '@/types/api'

const props = defineProps<{ category?: Category | null }>()
const emit = defineEmits<{ close: []; saved: [] }>()

const toast = useToast()
const busy = ref(false)
const error = ref<string | null>(null)

const form = reactive({
  name: props.category?.name ?? '',
  description: props.category?.description ?? '',
  icon: props.category?.icon ?? '',
  color: props.category?.color ?? '',
  sort_order: props.category?.sort_order ?? 0,
  is_visible: props.category?.is_visible ?? true,
})

async function submit(): Promise<void> {
  if (!form.name.trim()) {
    error.value = '分类名称不能为空'
    return
  }
  busy.value = true
  error.value = null
  try {
    if (props.category) {
      await categoriesApi.updateCategory(props.category.id, { ...form })
      toast.success('分类已更新')
    } else {
      await categoriesApi.createCategory({ ...form })
      toast.success('分类已创建')
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
  <ModalDialog :title="props.category ? '编辑分类' : '新建分类'" @close="emit('close')">
    <form @submit.prevent="submit">
      <div class="field">
        <label for="cat-name">名称</label>
        <input id="cat-name" v-model="form.name" class="input" maxlength="64" required />
      </div>
      <div class="field">
        <label for="cat-desc">描述（可选）</label>
        <input id="cat-desc" v-model="form.description" class="input" maxlength="512" />
      </div>
      <div class="field">
        <label for="cat-icon">图标（可选，emoji 或极短文字）</label>
        <input id="cat-icon" v-model="form.icon" class="input" maxlength="8" placeholder="例如 ✨ 或 常用" />
      </div>
      <div class="field">
        <label for="cat-color">强调色（可选）</label>
        <div class="row">
          <input id="cat-color" v-model="form.color" class="input" placeholder="#RRGGBB" />
          <input
            :value="form.color || '#6d5dfc'"
            type="color"
            style="width: 46px; height: 40px; padding: 2px; border-radius: 10px"
            aria-label="选择颜色"
            @input="form.color = ($event.target as HTMLInputElement).value"
          />
        </div>
      </div>
      <div class="field">
        <label for="cat-order">排序值（越小越靠前）</label>
        <input id="cat-order" v-model.number="form.sort_order" class="input" type="number" />
      </div>
      <label class="checkbox">
        <input v-model="form.is_visible" type="checkbox" />
        公开显示（关闭后仅管理员可见）
      </label>

      <p v-if="error" class="error-text">{{ error }}</p>

      <div class="modal__actions">
        <button type="button" class="btn" :disabled="busy" @click="emit('close')">取消</button>
        <button type="submit" class="btn btn--primary" :disabled="busy">
          {{ busy ? '保存中…' : '保存' }}
        </button>
      </div>
    </form>
  </ModalDialog>
</template>
