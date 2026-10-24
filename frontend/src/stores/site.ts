import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getNavigation } from '@/api/public'
import { applyTheme } from '@/composables/useTheme'
import type { Category, PublicSettings } from '@/types/api'

/** Public site content: settings and the navigation tree. */
export const useSiteStore = defineStore('site', () => {
  const settings = ref<PublicSettings | null>(null)
  const categories = ref<Category[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const loaded = ref(false)

  async function load(force = false): Promise<void> {
    if (loaded.value && !force) return
    loading.value = true
    error.value = null
    try {
      const nav = await getNavigation()
      settings.value = nav.settings
      categories.value = nav.categories
      loaded.value = true
      applyTheme(nav.settings.theme)
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e)
    } finally {
      loading.value = false
    }
  }

  return { settings, categories, loading, error, loaded, load }
})
