<script setup lang="ts">
import { reactive, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { errorMessage, useToast } from '@/composables/useToast'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const toast = useToast()

const form = reactive({ username: '', password: '' })
const busy = ref(false)
const error = ref<string | null>(null)

async function submit(): Promise<void> {
  busy.value = true
  error.value = null
  try {
    await auth.signIn(form.username, form.password)
    toast.success('登录成功')
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/admin'
    await router.replace(redirect)
  } catch (e) {
    error.value = errorMessage(e)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="page page--narrow">
    <div class="panel">
      <h1 style="margin-top: 0; font-size: 1.2rem">管理员登录</h1>
      <p class="dim" style="font-size: 0.86rem">登录后可管理分类、站点与系统设置。</p>

      <form @submit.prevent="submit">
        <div class="field">
          <label for="username">用户名</label>
          <input id="username" v-model="form.username" class="input" autocomplete="username" required />
        </div>
        <div class="field">
          <label for="password">密码</label>
          <input id="password" v-model="form.password" class="input" type="password" autocomplete="current-password" required />
        </div>

        <p v-if="error" class="error-text">{{ error }}</p>

        <button type="submit" class="btn btn--primary btn--block" :disabled="busy">
          {{ busy ? '登录中…' : '登录' }}
        </button>
      </form>

      <p class="dim" style="margin-bottom: 0; font-size: 0.84rem">
        <RouterLink class="badge" to="/">返回首页</RouterLink>
      </p>
    </div>
  </div>
</template>
