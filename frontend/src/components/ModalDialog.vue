<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'

const props = defineProps<{ title: string; width?: string }>()
const emit = defineEmits<{ close: [] }>()

const panel = ref<HTMLElement | null>(null)

function onKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape') emit('close')
}

onMounted(() => document.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => document.removeEventListener('keydown', onKeydown))
</script>

<template>
  <div class="modal-backdrop" @click.self="emit('close')">
    <div ref="panel" class="modal" :style="props.width ? { width: props.width } : undefined" role="dialog" aria-modal="true">
      <div class="modal__head">
        <h2 class="modal__title">{{ title }}</h2>
        <button type="button" class="btn btn--sm btn--ghost" aria-label="关闭" @click="emit('close')">✕</button>
      </div>
      <slot />
    </div>
  </div>
</template>
