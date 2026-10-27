<script setup lang="ts">
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
    <label class="theme-picker">
      <span class="sr-only">切换视觉主题</span>
      <select class="theme-picker__select" aria-label="切换视觉主题" :value="current" @change="changeTheme">
        <option value="dark">🌙 深色</option>
        <option value="light">☀️ 浅色</option>
        <option value="pond">🐟 锦鲤池塘</option>
      </select>
    </label>
    <template v-if="auth.authenticated">
      <RouterLink class="btn btn--sm btn--primary" to="/admin">管理后台</RouterLink>
      <button type="button" class="btn btn--sm" @click="signOut">退出</button>
    </template>
    <RouterLink v-else class="btn btn--sm" to="/login">登录</RouterLink>
  </div>
</template>
