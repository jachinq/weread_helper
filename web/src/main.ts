import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import { router } from './router'
import { bootTheme } from './themes/apply'
import './style.css'

bootTheme()
createApp(App).use(createPinia()).use(router).mount('#app')
