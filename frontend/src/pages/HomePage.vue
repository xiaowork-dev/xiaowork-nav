<script setup lang="ts">
import { onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import AppHeader from '@/components/AppHeader.vue'
import ScrollLayout from '@/components/ScrollLayout.vue'
import { useSiteStore } from '@/stores/site'

/**
 * Public home page.
 *
 * The page itself only orchestrates: it loads the shared navigation data and
 * renders the selected layout. Both layouts read the same store, so switching
 * never duplicates or diverges the data.
 */
const site = useSiteStore()

onMounted(() => {
  void site.load()
})
</script>

<template>
  <div class="page">
    <AppHeader />

    <div v-if="site.loading" class="empty">
      <span class="spinner" style="margin: 0 auto 10px" />
      正在加载导航数据…
    </div>

    <div v-else-if="site.error" class="empty">
      <p>无法加载导航数据：{{ site.error }}</p>
      <p class="dim">
        若管理员已关闭游客访问，请先
        <RouterLink class="badge" to="/login">登录</RouterLink>
      </p>
    </div>

    <ScrollLayout v-else />
  </div>
</template>
