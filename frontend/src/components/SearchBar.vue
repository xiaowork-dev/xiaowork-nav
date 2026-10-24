<script setup lang="ts">
import { ref, watch } from 'vue'
import type { SearchEngine } from '@/types/api'

const props = defineProps<{
  engines: SearchEngine[]
  defaultEngine: string
  modelValue: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
  submit: [engine: SearchEngine]
}>()

const engineName = ref(props.defaultEngine || props.engines[0]?.name || '')

watch(
  () => props.defaultEngine,
  (value) => {
    if (value) engineName.value = value
  },
)

function onInput(event: Event): void {
  emit('update:modelValue', (event.target as HTMLInputElement).value)
}

function currentEngine(): SearchEngine | undefined {
  return props.engines.find((e) => e.name === engineName.value) ?? props.engines[0]
}

function onSubmit(): void {
  const engine = currentEngine()
  if (engine) emit('submit', engine)
}
</script>

<template>
  <div class="search">
    <form class="search__box" role="search" @submit.prevent="onSubmit">
      <select v-model="engineName" class="search__engine" aria-label="搜索引擎">
        <option v-for="engine in engines" :key="engine.name" :value="engine.name">{{ engine.name }}</option>
      </select>
      <input
        class="search__input"
        type="search"
        :value="modelValue"
        placeholder="输入关键词：实时筛选站点，回车用所选引擎搜索"
        aria-label="搜索"
        @input="onInput"
      />
      <button type="submit" class="btn btn--sm btn--primary">搜索</button>
    </form>
  </div>
</template>
