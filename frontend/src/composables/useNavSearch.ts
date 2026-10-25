import { computed, onUnmounted, ref, watch } from 'vue'
import { searchBookmarks } from '@/api/public'
import { useSiteStore } from '@/stores/site'
import type { Bookmark, SearchEngine } from '@/types/api'

/**
 * Search state shared by both navigation layouts.
 *
 * Each layout mounts this composable once, so the two layouts cannot diverge in
 * behaviour: same API call, same local fallback, same debounce.
 */
export function useNavSearch() {
  const site = useSiteStore()
  const query = ref('')
  const results = ref<Bookmark[]>([])
  const searching = ref(false)
  let timer: number | undefined

  const engines = computed<SearchEngine[]>(() => site.settings?.search_engines ?? [])
  const defaultEngine = computed(() => site.settings?.default_search_engine ?? '')
  const showResults = computed(() => query.value.trim().length > 0)

  watch(query, (value) => {
    window.clearTimeout(timer)
    const q = value.trim()
    if (!q) {
      results.value = []
      searching.value = false
      return
    }
    searching.value = true
    timer = window.setTimeout(async () => {
      try {
        results.value = (await searchBookmarks(q)).items
      } catch {
        // Fall back to a local filter when the API is unavailable.
        const lowered = q.toLowerCase()
        results.value = site.categories
          .flatMap((c) => c.bookmarks ?? [])
          .filter((b) =>
            [b.title, b.description, b.keywords].some((field) => field.toLowerCase().includes(lowered)),
          )
      } finally {
        searching.value = false
      }
    }, 250)
  })

  /** Opens the query in the given search engine. */
  function openEngine(engine: SearchEngine): void {
    const q = query.value.trim()
    if (!q) return
    const url = engine.url_template.replace('{query}', encodeURIComponent(q))
    window.open(url, '_blank', 'noopener')
  }

  function reset(): void {
    query.value = ''
  }

  onUnmounted(() => window.clearTimeout(timer))

  return { query, results, searching, engines, defaultEngine, showResults, openEngine, reset }
}
