<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { errorMessage, useToast } from '@/composables/useToast'

const auth = useAuthStore()
const router = useRouter()
const toast = useToast()

const form = reactive({ username: 'admin', password: '', confirm: '' })
const busy = ref(false)
const error = ref<string | null>(null)

async function submit(): Promise<void> {
  if (form.password.length < 8) {
    error.value = '密码至少 8 个字符'
    return
  }
  if (form.password !== form.confirm) {
    error.value = '两次输入的密码不一致'
    return
  }
  busy.value = true
  error.value = null
  try {
    await auth.setupAdmin(form.username, form.password)
    toast.success('管理员已创建')
    await router.replace('/admin')
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
      <h1 style="margin-top: 0; font-size: 1.2rem">初始化 XiaoWork Nav</h1>
      <p class="dim" style="font-size: 0.86rem">
        首次使用请创建唯一的管理员账号。创建后初始化入口将自动关闭。
      </p>

      <form @submit.prevent="submit">
        <div class="field">
          <label for="su-username">用户名</label>
          <input id="su-username" v-model="form.username" class="input" minlength="3" maxlength="32" required />
          <span class="hint">3–32 位，仅限字母、数字、下划线与连字符</span>
        </div>
        <div class="field">
          <label for="su-password">密码</label>
          <input id="su-password" v-model="form.password" class="input" type="password" minlength="8" required />
        </div>
        <div class="field">
          <label for="su-confirm">确认密码</label>
          <input id="su-confirm" v-model="form.confirm" class="input" type="password" minlength="8" required />
        </div>

        <p v-if="error" class="error-text">{{ error }}</p>

        <button type="submit" class="btn btn--primary btn--block" :disabled="busy">
          {{ busy ? '创建中…' : '创建管理员并进入后台' }}
        </button>
      </form>
    </div>
  </div>
</template>
