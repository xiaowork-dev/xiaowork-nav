<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { IconType } from '@/types/api'

const props = defineProps<{
  type: IconType
  value: string
  title: string
}>()

const failed = ref(false)
watch(
  () => [props.type, props.value],
  () => {
    failed.value = false
  },
)

const src = computed(() => (props.type !== 'none' && props.value && !failed.value ? props.value : ''))
const letter = computed(() => (props.title.trim().charAt(0) || '?').toUpperCase())
</script>

<template>
  <img
    v-if="src"
    :src="src"
    :alt="title"
    loading="lazy"
    referrerpolicy="no-referrer"
    @error="failed = true"
  />
  <span v-else class="tile__fallback">{{ letter }}</span>
</template>
