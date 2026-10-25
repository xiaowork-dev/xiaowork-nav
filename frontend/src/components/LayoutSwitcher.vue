<script setup lang="ts">
import { useLayout } from '@/composables/useLayout'
import type { NavLayout } from '@/composables/useLayout'

/**
 * One-click switch between the two navigation layouts.
 *
 * It only touches the layout preference — the theme is a separate concern and
 * is never modified here.
 */
const { current, setLayout } = useLayout()

const options: Array<{ value: NavLayout; label: string; icon: string; title: string }> = [
  { value: 'scroll', label: '滚动模式', icon: '☰', title: '滚动模式：所有分类纵向排列，PC 端带左侧分类栏' },
  { value: 'tabs', label: '标签模式', icon: '▤', title: '标签模式：顶部分类标签，一次只看一个分类' },
]
</script>

<template>
  <div class="layout-switch" role="group" aria-label="导航布局">
    <button
      v-for="option in options"
      :key="option.value"
      type="button"
      class="layout-switch__item"
      :class="{ 'is-active': current === option.value }"
      :title="option.title"
      :aria-pressed="current === option.value"
      @click="setLayout(option.value)"
    >
      <span aria-hidden="true">{{ option.icon }}</span>
      <span class="layout-switch__label">{{ option.label }}</span>
    </button>
  </div>
</template>
