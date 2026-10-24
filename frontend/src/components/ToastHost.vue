<script setup lang="ts">
import { useToast } from '@/composables/useToast'

const { toasts, dismiss } = useToast()
</script>

<template>
  <div class="toasts">
    <div
      v-for="toast in toasts"
      :key="toast.id"
      class="toast"
      :class="`toast--${toast.kind}`"
      role="status"
      @click="dismiss(toast.id)"
    >
      {{ toast.message }}
    </div>
  </div>
</template>
