<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import AppHeader from '@/components/AppHeader.vue'
import ScrollLayout from '@/components/ScrollLayout.vue'
import TabsLayout from '@/components/TabsLayout.vue'
import { useLayout } from '@/composables/useLayout'
import { useSiteStore } from '@/stores/site'

/**
 * Public home page.
 *
 * The page only orchestrates: it loads the shared navigation data and renders
 * the selected layout. Both layouts read the same store and the same API, so
 * switching never duplicates or diverges the data, and the theme is untouched.
 *
 * The scroll layout uses a full-bleed shell (`page--rail`) so its category rail
 * can sit on the real page edge; the tabs layout keeps the centred container.
 */
const site = useSiteStore()
const { current } = useLayout()

const isTabs = computed(() => current.value === 'tabs')

onMounted(() => {
  void site.load()
})
</script>

<template>
  <div class="page" :class="isTabs ? 'page--tabs' : 'page--rail'">
    <AppHeader show-layout-switcher />

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
