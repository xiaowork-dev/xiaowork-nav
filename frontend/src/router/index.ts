import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'home',
      component: () => import('@/pages/HomePage.vue'),
    },
    {
      path: '/login',
      name: 'login',
      component: () => import('@/pages/LoginPage.vue'),
      meta: { anonymous: true },
    },
    {
      path: '/setup',
      name: 'setup',
      component: () => import('@/pages/SetupPage.vue'),
      meta: { anonymous: true },
    },
    {
      path: '/admin',
      component: () => import('@/pages/admin/AdminLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        { path: '', redirect: { name: 'admin-categories' } },
        { path: 'categories', name: 'admin-categories', component: () => import('@/pages/admin/CategoriesPage.vue') },
        { path: 'bookmarks', name: 'admin-bookmarks', component: () => import('@/pages/admin/BookmarksPage.vue') },
        { path: 'settings', name: 'admin-settings', component: () => import('@/pages/admin/SettingsPage.vue') },
        { path: 'data', name: 'admin-data', component: () => import('@/pages/admin/DataPage.vue') },
      ],
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: () => import('@/pages/NotFoundPage.vue'),
    },
  ],
  scrollBehavior() {
    return { top: 0 }
  },
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (!auth.loaded) {
    await auth.refresh()
  }

  // First run: force the setup wizard.
  if (to.name !== 'setup' && !auth.initialized) {
    return { name: 'setup' }
  }
  if (to.name === 'setup' && auth.initialized) {
    return { name: auth.authenticated ? 'admin-categories' : 'login' }
  }
  if (to.meta.requiresAuth && !auth.authenticated) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (to.name === 'login' && auth.authenticated) {
    return { name: 'admin-categories' }
  }
  return true
})

export default router
