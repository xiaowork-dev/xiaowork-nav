<script setup lang="ts">
import ModalDialog from './ModalDialog.vue'

defineProps<{
  title?: string
  message: string
  confirmLabel?: string
  busy?: boolean
}>()

const emit = defineEmits<{ confirm: []; cancel: [] }>()
</script>

<template>
  <ModalDialog :title="title ?? '请确认'" @close="emit('cancel')">
    <p class="muted" style="margin-top: 0">{{ message }}</p>
    <div class="modal__actions">
      <button type="button" class="btn" :disabled="busy" @click="emit('cancel')">取消</button>
      <button type="button" class="btn btn--danger" :disabled="busy" @click="emit('confirm')">
        {{ confirmLabel ?? '删除' }}
      </button>
    </div>
  </ModalDialog>
</template>
