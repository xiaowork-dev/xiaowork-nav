<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import * as adminApi from '@/api/admin'
import * as authApi from '@/api/auth'
import { applyTheme } from '@/composables/useTheme'
import { errorMessage, useToast } from '@/composables/useToast'
import { useSiteStore } from '@/stores/site'
import type { AdminSettings, SearchEngine } from '@/types/api'

const toast = useToast()
const site = useSiteStore()

const loading = ref(false)
const saving = ref(false)
const passwordBusy = ref(false)

const form = reactive<AdminSettings>({
  site_title: '',
  site_favicon: '',
  theme: 'auto',
  search_engines: [],
  default_search_engine: '',
  open_in_new_tab_default: true,
  homepage_layout: 'grid',
  allow_guest_access: true,
})

const password = reactive({ current: '', next: '', confirm: '' })

async function load(): Promise<void> {
  loading.value = true
  try {
    const settings = await adminApi.getAdminSettings()
    Object.assign(form, settings)
    if (!form.search_engines) form.search_engines = []
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    loading.value = false
  }
}

function addEngine(): void {
  form.search_engines.push({ name: '', url_template: 'https://example.com/search?q={query}' })
}

function removeEngine(index: number): void {
  form.search_engines.splice(index, 1)
}

async function save(): Promise<void> {
  saving.value = true
  try {
    const engines: SearchEngine[] = form.search_engines
      .map((e) => ({ name: e.name.trim(), url_template: e.url_template.trim() }))
      .filter((e) => e.name && e.url_template)

    const updated = await adminApi.updateAdminSettings({
      site_title: form.site_title,
      site_favicon: form.site_favicon,
      theme: form.theme,
      homepage_layout: form.homepage_layout,
      open_in_new_tab_default: form.open_in_new_tab_default,
      allow_guest_access: form.allow_guest_access,
      search_engines: engines,
      default_search_engine: form.default_search_engine,
    })
    Object.assign(form, updated)
    applyTheme(updated.theme)
    // Refresh shared public content so the header reflects the new title.
    await site.load(true)
    toast.success('设置已保存')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    saving.value = false
  }
}

async function changePassword(): Promise<void> {
  if (password.next.length < 8) {
    toast.error('新密码至少 8 个字符')
    return
  }
  if (password.next !== password.confirm) {
    toast.error('两次输入的新密码不一致')
    return
  }
  passwordBusy.value = true
  try {
    await authApi.changePassword(password.current, password.next)
    toast.success('密码已修改，请重新登录')
    password.current = ''
    password.next = ''
    password.confirm = ''
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    passwordBusy.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="card">
    <div class="card__head">
      <h2 class="card__title">系统设置</h2>
    </div>

    <div v-if="loading" class="empty">加载中…</div>

    <form v-else @submit.prevent="save">
      <div class="field">
        <label for="s-title">站点标题</label>
        <input id="s-title" v-model="form.site_title" class="input" maxlength="128" />
      </div>

      <div class="field">
        <label for="s-favicon">站点 Favicon 地址（可选）</label>
        <input id="s-favicon" v-model="form.site_favicon" class="input" placeholder="https://… 或 /api/v1/public/uploads/…" />
      </div>

      <div class="field">
        <label for="s-theme">默认主题</label>
        <select id="s-theme" v-model="form.theme" class="select">
          <option value="auto">跟随系统</option>
          <option value="light">浅色</option>
          <option value="dark">深色</option>
        </select>
        <span class="hint">访客本地切换优先于该设置。</span>
      </div>

      <div class="field">
        <label for="s-layout">首页布局</label>
        <select id="s-layout" v-model="form.homepage_layout" class="select">
          <option value="grid">网格</option>
          <option value="list">列表</option>
        </select>
      </div>

      <label class="checkbox" style="margin-bottom: 12px">
        <input v-model="form.open_in_new_tab_default" type="checkbox" />
        站点默认在新标签页打开
      </label>
      <br />
      <label class="checkbox" style="margin-bottom: 16px">
        <input v-model="form.allow_guest_access" type="checkbox" />
        允许游客浏览公开导航（关闭后需登录才能访问首页）
      </label>

      <div class="field">
        <label>搜索引擎</label>
        <div class="list">
          <div v-for="(engine, index) in form.search_engines" :key="index" class="list-item">
            <input v-model="engine.name" class="input" style="max-width: 150px" placeholder="名称" />
            <input v-model="engine.url_template" class="input" placeholder="https://…/search?q={query}" />
            <button type="button" class="btn btn--sm btn--danger" @click="removeEngine(index)">删除</button>
          </div>
        </div>
        <button type="button" class="btn btn--sm" style="margin-top: 8px" @click="addEngine">添加搜索引擎</button>
        <span class="hint">模板必须包含 <code class="code">{query}</code> 占位符。</span>
      </div>

      <div class="field">
        <label for="s-default-engine">默认搜索引擎</label>
        <select id="s-default-engine" v-model="form.default_search_engine" class="select">
          <option v-for="engine in form.search_engines" :key="engine.name" :value="engine.name">
            {{ engine.name || '（未命名）' }}
          </option>
        </select>
      </div>

      <button type="submit" class="btn btn--primary" :disabled="saving">
        {{ saving ? '保存中…' : '保存设置' }}
      </button>
    </form>
  </div>

  <div class="card">
    <div class="card__head">
      <h2 class="card__title">修改密码</h2>
    </div>
    <form @submit.prevent="changePassword">
      <div class="field">
        <label for="p-current">当前密码</label>
        <input id="p-current" v-model="password.current" class="input" type="password" autocomplete="current-password" required />
      </div>
      <div class="field">
        <label for="p-next">新密码</label>
        <input id="p-next" v-model="password.next" class="input" type="password" minlength="8" autocomplete="new-password" required />
      </div>
      <div class="field">
        <label for="p-confirm">确认新密码</label>
        <input id="p-confirm" v-model="password.confirm" class="input" type="password" minlength="8" autocomplete="new-password" required />
      </div>
      <p class="hint">修改密码后所有登录会话都会失效，需要重新登录。</p>
      <button type="submit" class="btn" :disabled="passwordBusy">
        {{ passwordBusy ? '提交中…' : '修改密码' }}
      </button>
    </form>
  </div>
</template>
