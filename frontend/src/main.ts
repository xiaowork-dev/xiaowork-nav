import { createApp } from 'vue'
import { createPinia } from 'pinia'

import App from './App.vue'
import router from './router'
import { applyTheme } from './composables/useTheme'
import './styles/main.css'
import './styles/pond-theme.css'

// Apply an initial theme (system preference or a local override) before the
// site settings are fetched.
applyTheme('auto')

createApp(App).use(createPinia()).use(router).mount('#app')
