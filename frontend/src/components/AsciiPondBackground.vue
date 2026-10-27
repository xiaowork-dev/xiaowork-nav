<script setup lang="ts">
import { onUnmounted, watch } from 'vue'
import { useTheme } from '@/composables/useTheme'
const { current } = useTheme()
let destroyed = false
let instance: { destroy: () => void; canvas: HTMLCanvasElement } | undefined
let generation = 0

// Do not download or execute the optional engine for visitors using classic themes.
watch(current, async (theme) => {
  const token = ++generation
  instance?.destroy()
  instance = undefined
  if (theme !== 'pond') return
  try {
    // Vendored upstream ES modules: no CDN and no additional runtime dependency.
    const modulePath = '/vendor/ascii-pond/pond.js'
    const { mount } = await import(/* @vite-ignore */ modulePath)
    if (destroyed || token !== generation || current.value !== 'pond') return
    const result = mount(document.body, {
      theme: 'dark',
      fixed: true,
      species: [{ type: 'koi', count: 4, young: 0 }],
      avoid: '.site-header, .rail, .shell__topbar, .clock, .search, .category__head, .tiles, .tabs, .tabs-panel__head, .admin',
      ignore: 'a, button, input, select, textarea, label, .rail, .tile, .tabs, .search, .site-header, .shell__topbar',
      palette: { dark: { page: [26, 25, 23], bg: [26, 25, 33] } },
    }) as { destroy: () => void; canvas: HTMLCanvasElement }
    result.canvas.classList.add('ascii-pond-canvas')
    instance = result
  } catch (error) {
    console.warn('ASCII Pond background could not start:', error)
  }
}, { immediate: true })

onUnmounted(() => {
  destroyed = true
  ++generation
  instance?.destroy()
  instance = undefined
})
</script>

<template></template>
