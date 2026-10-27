<script setup lang="ts">
import { onUnmounted, watch } from 'vue'
import { useTheme } from '@/composables/useTheme'
import { createPondController, type PondModule, type PondOptions } from '@/composables/useAsciiPond'

const { current } = useTheme()

/** Vendored upstream ES module: no CDN, no extra runtime dependency. */
function load(): Promise<PondModule> {
  const modulePath = '/vendor/ascii-pond/pond.js'
  return import(/* @vite-ignore */ modulePath) as Promise<PondModule>
}

/**
 * Phones get one koi fewer: the pond is decoration, the navigation is the page.
 * The engine keeps its own creatures in proportion to the viewport anyway.
 */
function options(): PondOptions {
  return {
    theme: 'dark',
    fixed: true,
    species: [{ type: 'koi', count: window.innerWidth < 700 ? 3 : 4, young: 0 }],
    avoid:
      '.site-header, .rail, .shell__topbar, .clock, .search, .category__head, .tiles, .tabs, .tabs-panel__head, .admin',
    ignore:
      'a, button, input, select, textarea, label, [data-pond-ignore], .rail, .tile, .tabs, .search, .site-header, .shell__topbar',
    palette: { dark: { page: [26, 25, 23], bg: [26, 25, 33] } },
  }
}

const pond = createPondController({
  load,
  target: () => document.body,
  options,
  onMount: (instance) => instance.canvas?.classList.add('ascii-pond-canvas'),
})

// Do not download or execute the optional engine for visitors using classic themes.
watch(current, (theme) => pond.sync(theme), { immediate: true })

onUnmounted(() => pond.dispose())
</script>

<template></template>
