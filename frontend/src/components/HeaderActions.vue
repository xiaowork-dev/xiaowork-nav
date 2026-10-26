<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import LayoutSwitcher from '@/components/LayoutSwitcher.vue'
import { useAuthStore } from '@/stores/auth'
import { useToast, errorMessage } from '@/composables/useToast'
import { useTheme } from '@/composables/useTheme'

/**
 * Top-right actions, shared by both shells so the controls and their styling
 * cannot drift apart: the tabs layout shows them in the page header, the scroll
 * layout shows them in the shell's top bar.
 */
withDefaults(defineProps<{ showLayoutSwitcher?: boolean }>(), {
  showLayoutSwitcher: false,
})

const auth = useAuthStore()
const router = useRouter()
const toast = useToast()
const { current, toggleTheme } = useTheme()

const themeLabel = computed(() => (current.value === 'dark' ? '深色' : '浅色'))

async function signOut(): Promise<void> {
  try {
    await auth.signOut()
    toast.success('已退出登录')
    await router.push({ name: 'home' })
  } catch (error) {
    toast.error(errorMessage(error))
  }
}
</script>

<template>
  <div class="header-actions">
    <LayoutSwitcher v-if="showLayoutSwitcher" />
    <button type="button" class="btn btn--sm" :title="`切换主题（当前：${themeLabel}）`" @click="toggleTheme">
      {{ current === 'dark' ? '🌙' : '☀️' }} {{ themeLabel }}
    </button>
    <template v-if="auth.authenticated">
      <RouterLink class="btn btn--sm btn--primary" to="/admin">管理后台</RouterLink>
      <button type="button" class="btn btn--sm" @click="signOut">退出</button>
    </template>
    <RouterLink v-else class="btn btn--sm" to="/login">登录</RouterLink>
  </div>
</template>
