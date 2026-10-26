<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { useSiteStore } from '@/stores/site'

/**
 * Site identity (mark + wordmark). Rendered in the rail on desktop and in the
 * top bar on phones, so both shells present the same brand element.
 *
 * The mark always renders: when no favicon is configured it falls back to the
 * first letter of the title on the brand gradient, which keeps the collapsed
 * rail from showing an empty strip.
 */
const site = useSiteStore()

const title = computed(() => site.settings?.site_title || 'XiaoWork Nav')
const favicon = computed(() => site.settings?.site_favicon || '')
const initial = computed(() => (title.value.trim().charAt(0) || 'X').toUpperCase())
</script>

<template>
  <RouterLink class="brand" to="/">
    <img v-if="favicon" :src="favicon" alt="" />
    <span v-else class="brand__mark" aria-hidden="true">{{ initial }}</span>
    <span class="brand__title">{{ title }}</span>
  </RouterLink>
</template>
