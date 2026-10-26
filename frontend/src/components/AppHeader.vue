<script setup lang="ts">
import BrandMark from '@/components/BrandMark.vue'
import HeaderActions from '@/components/HeaderActions.vue'

/**
 * Page header for the tabs layout (and for the scroll layout's loading/error
 * states): brand on the left, actions on the right, inside the centred page
 * container.
 */
withDefaults(defineProps<{ showLayoutSwitcher?: boolean }>(), {
  showLayoutSwitcher: false,
})
</script>

<template>
  <header class="site-header">
    <BrandMark />
    <HeaderActions :show-layout-switcher="showLayoutSwitcher" />
  </header>
</template>
