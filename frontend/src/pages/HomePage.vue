<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { getAuthStatus, getVersion } from '@/api/system'
import { ApiError } from '@/api/client'
import type { AuthStatus, VersionInfo } from '@/types/api'

const version = ref<VersionInfo | null>(null)
const auth = ref<AuthStatus | null>(null)
const error = ref<string | null>(null)
const loading = ref(true)

onMounted(async () => {
  try {
    const [v, a] = await Promise.all([getVersion(), getAuthStatus()])
    version.value = v
    auth.value = a
  } catch (e) {
    error.value = e instanceof ApiError ? `${e.code}: ${e.message}` : String(e)
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="page">
    <header class="hero">
      <h1 class="hero-title">XiaoWork Nav</h1>
      <p class="muted">个人导航站 · 前端骨架（P1）</p>
    </header>

    <section class="panel search">
      <input class="search-input" type="search" placeholder="搜索站点…（搜索功能将在后续阶段接入）" disabled />
    </section>

    <section v-if="loading" class="panel status">正在连接后端…</section>
    <section v-else-if="error" class="panel status error-text">后端连接失败：{{ error }}</section>
    <section v-else class="panel status">
      <p>
        <span class="badge">后端在线</span>
        版本 <code>{{ version?.version }}</code>
        · 数据库结构版本 <code>{{ version?.schema_version }}</code>
      </p>
      <p class="muted">
        初始化状态：{{ auth?.initialized ? '已初始化' : '未初始化（等待首次设置向导）' }}
        · 登录状态：{{ auth?.authenticated ? '已登录' : '游客' }}
      </p>
    </section>

    <section class="placeholder">
      <h2 class="placeholder-title">分类导航区</h2>
      <p class="muted">
        此处将按分类展示图标卡片（数据来自 <code>GET /api/v1/public/navigation</code>），
        在 P3 阶段实现。
      </p>
      <div class="tiles">
        <div v-for="n in 8" :key="n" class="tile">
          <div class="tile-icon" />
          <div class="tile-label" />
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.hero {
  padding: 24px 0 18px;
  text-align: center;
}

.hero-title {
  margin: 0;
  font-size: clamp(1.6rem, 3vw, 2.2rem);
  font-weight: 800;
  letter-spacing: 0.02em;
}

.search {
  width: min(100%, 760px);
  margin: 0 auto 32px;
  padding: 10px;
}

.search-input {
  width: 100%;
  height: 46px;
  padding: 0 16px;
  border: 1px solid var(--glass-border);
  border-radius: var(--radius);
  background: rgba(20, 24, 67, 0.28);
  color: var(--text-main);
  font-size: 0.98rem;
}

.search-input::placeholder {
  color: var(--text-dim);
}

.status {
  padding: 16px 20px;
  margin-bottom: 32px;
  font-size: 0.92rem;
}

.status p {
  margin: 4px 0;
}

code {
  padding: 1px 6px;
  border-radius: 6px;
  background: rgba(0, 0, 0, 0.22);
  font-size: 0.85em;
}

.placeholder-title {
  font-size: 1.1rem;
  font-weight: 700;
  margin: 0 0 6px;
}

.tiles {
  display: grid;
  grid-template-columns: repeat(auto-fill, var(--tile-size));
  gap: 26px 34px;
  margin-top: 22px;
}

.tile {
  width: var(--tile-size);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 7px;
}

.tile-icon {
  width: var(--tile-size);
  height: var(--tile-size);
  border-radius: var(--radius-sm);
  background: rgba(255, 255, 255, 0.22);
  border: 1px solid var(--glass-border);
}

.tile-label {
  width: 46px;
  height: 8px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.2);
}
</style>
