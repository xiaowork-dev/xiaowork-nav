<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import AppHeader from '@/components/AppHeader.vue'
import AsciiPondBackground from '@/components/AsciiPondBackground.vue'
import ScrollLayout from '@/components/ScrollLayout.vue'
import TabsLayout from '@/components/TabsLayout.vue'
import { useLayout } from '@/composables/useLayout'
import { useSiteStore } from '@/stores/site'

const site = useSiteStore()
const { current } = useLayout()
const isTabs = computed(() => current.value === 'tabs')
const showHeader = computed(() => isTabs.value || site.loading || Boolean(site.error))
onMounted(() => { void site.load() })
</script>

<template>
  <AsciiPondBackground />
  <div class="page" :class="isTabs ? 'page--tabs' : 'page--rail'">
    <AppHeader v-if="showHeader" show-layout-switcher />
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
    <TabsLayout v-else-if="isTabs" />
    <ScrollLayout v-else />
  </div>
</template>
