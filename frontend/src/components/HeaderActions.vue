<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import LayoutSwitcher from '@/components/LayoutSwitcher.vue'
import { useAuthStore } from '@/stores/auth'
import { useToast, errorMessage } from '@/composables/useToast'
import { useTheme, type VisualTheme } from '@/composables/useTheme'

withDefaults(defineProps<{ showLayoutSwitcher?: boolean }>(), {
  showLayoutSwitcher: false,
})
const auth = useAuthStore()
const router = useRouter()
const toast = useToast()
const { current, setTheme } = useTheme()

/**
 * Phone headers have no spare width: a 102px select pushed the top bar past the
 * viewport edge. Short labels keep the same three choices in ~55px.
 */
const compact = ref(false)
let query: MediaQueryList | undefined
function sync(event: MediaQueryList | MediaQueryListEvent): void {
  compact.value = event.matches
}
onMounted(() => {
  query = window.matchMedia('(max-width: 700px)')
  sync(query)
  query.addEventListener('change', sync)
})
onUnmounted(() => query?.removeEventListener('change', sync))

async function signOut(): Promise<void> {
  try {
    await auth.signOut()
    toast.success('已退出登录')
    await router.push({ name: 'home' })
  } catch (error) {
    toast.error(errorMessage(error))
  }
}
function changeTheme(event: Event) {
  const value = (event.target as HTMLSelectElement).value
  if (value === 'dark' || value === 'light' || value === 'pond') setTheme(value as VisualTheme)
}
</script>

<template>
  <div class="header-actions">
    <LayoutSwitcher v-if="showLayoutSwitcher" />
    <select class="theme-picker__select" aria-label="切换视觉主题" :value="current" @change="changeTheme">
      <option value="dark">{{ compact ? '深色' : '🌙 深色' }}</option>
      <option value="light">{{ compact ? '浅色' : '☀️ 浅色' }}</option>
      <option value="pond">{{ compact ? '池塘' : '🐟 锦鲤池塘' }}</option>
    </select>
    <template v-if="auth.authenticated">
      <RouterLink class="btn btn--sm btn--primary" to="/admin">管理后台</RouterLink>
      <button type="button" class="btn btn--sm" @click="signOut">退出</button>
    </template>
    <RouterLink v-else class="btn btn--sm" to="/login">登录</RouterLink>
  </div>
</template>
