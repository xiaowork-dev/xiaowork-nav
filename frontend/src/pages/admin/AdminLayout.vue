<script setup lang="ts">
import { RouterLink, RouterView } from 'vue-router'
import AppHeader from '@/components/AppHeader.vue'
</script>

<template>
  <div>
    <div style="width: min(100%, 1180px); margin: 0 auto; padding: 20px clamp(14px, 3vw, 32px) 0">
      <AppHeader />
    </div>

    <div class="admin">
      <nav class="admin__nav">
        <RouterLink to="/admin/categories">分类管理</RouterLink>
        <RouterLink to="/admin/bookmarks">网站管理</RouterLink>
        <RouterLink to="/admin/settings">系统设置</RouterLink>
        <RouterLink to="/admin/data">数据管理</RouterLink>
        <RouterLink to="/">返回首页</RouterLink>
      </nav>
      <div class="admin__content">
        <RouterView />
      </div>
    </div>
  </div>
</template>
