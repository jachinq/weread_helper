<script setup lang="ts">
import { RouterLink, useRoute } from 'vue-router'
import { navClassFor } from './useAppChrome'

const route = useRoute()

function navClass(path: string, prefix?: boolean) {
  return navClassFor(path, route.path, prefix)
}
</script>

<template>
  <nav class="site-nav dock" aria-label="移动端导航">
    <RouterLink to="/" :class="navClass('/')" :aria-current="route.path === '/' ? 'page' : undefined">
      <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 11.5 12 4l8 7.5V20a1 1 0 0 1-1 1h-5v-6H10v6H5a1 1 0 0 1-1-1z" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linejoin="round"/></svg>
      首页
    </RouterLink>
    <RouterLink
      to="/notes"
      :class="navClass('/notes', true)"
      :aria-current="route.path.startsWith('/notes') ? 'page' : undefined"
    >
      <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M7 4h8l4 4v12H7z" fill="none" stroke="currentColor" stroke-width="1.7"/><path d="M15 4v4h4M9 12h8M9 16h6" fill="none" stroke="currentColor" stroke-width="1.7"/></svg>
      笔记
    </RouterLink>
    <RouterLink
      to="/shelf"
      :class="navClass('/shelf', true)"
      :aria-current="route.path.startsWith('/shelf') ? 'page' : undefined"
    >
      <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 19h16M6 19V7h4v12M12 19V5h4v14M18 19v-8h2v8" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linejoin="round"/></svg>
      书架
    </RouterLink>
    <RouterLink
      to="/stats"
      :class="navClass('/stats', true)"
      :aria-current="route.path.startsWith('/stats') ? 'page' : undefined"
    >
      <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M5 19V10h3v9H5zm6 0V5h3v14h-3zm6 0v-7h3v7h-3z" fill="none" stroke="currentColor" stroke-width="1.7"/></svg>
      统计
    </RouterLink>
    <RouterLink
      to="/settings"
      :class="navClass('/settings', true)"
      :aria-current="route.path.startsWith('/settings') ? 'page' : undefined"
    >
      <svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="3.2" fill="none" stroke="currentColor" stroke-width="1.7"/><path d="M12 3.5v2.2M12 18.3v2.2M3.5 12h2.2M18.3 12h2.2M6.1 6.1l1.6 1.6M16.3 16.3l1.6 1.6M17.9 6.1l-1.6 1.6M7.7 16.3l-1.6 1.6" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round"/></svg>
      设置
    </RouterLink>
  </nav>
</template>
