<script setup lang="ts">
import { RouterLink, RouterView } from 'vue-router'
import { siteTitle } from '../../site'
import { themeHasLamp, themeKicker } from '../apply'
import DockNav from '../DockNav.vue'
import { useAppChrome } from '../useAppChrome'

const { route, status, syncingClick, navClass, onSync, formatSyncTime } = useAppChrome()
</script>

<template>
  <div class="shell shell-sidebar">
    <a class="skip-link" href="#main">跳到正文</a>
    <div v-if="themeHasLamp" class="lamp" aria-hidden="true" />

    <aside class="rail" aria-label="站点栏">
      <RouterLink class="brand" to="/" :aria-label="siteTitle + '首页'">
        <span class="seal" aria-hidden="true">读</span>
        <div>
          <p class="kicker">{{ themeKicker }}</p>
          <h1>{{ siteTitle }}</h1>
        </div>
      </RouterLink>
      <nav class="site-nav rail-nav" aria-label="主导航">
        <RouterLink to="/" :class="navClass('/')" :aria-current="route.path === '/' ? 'page' : undefined">首页</RouterLink>
        <RouterLink
          to="/notes"
          :class="navClass('/notes', true)"
          :aria-current="route.path.startsWith('/notes') ? 'page' : undefined"
        >笔记</RouterLink>
        <RouterLink
          to="/shelf"
          :class="navClass('/shelf', true)"
          :aria-current="route.path.startsWith('/shelf') ? 'page' : undefined"
        >书架</RouterLink>
        <RouterLink
          to="/stats"
          :class="navClass('/stats', true)"
          :aria-current="route.path.startsWith('/stats') ? 'page' : undefined"
        >统计</RouterLink>
        <RouterLink
          to="/settings"
          :class="navClass('/settings', true)"
          :aria-current="route.path.startsWith('/settings') ? 'page' : undefined"
        >设置</RouterLink>
      </nav>
      <div class="sync-box rail-sync">
        <span class="sync-meta">{{ formatSyncTime(status?.lastOkAt || 0) }}</span>
        <button
          class="btn btn-solid"
          type="button"
          :disabled="status?.state === 'running' || syncingClick"
          :aria-busy="status?.state === 'running' || syncingClick"
          @click="onSync"
        >
          {{ status?.state === 'running' || syncingClick ? '同步中…' : '同步' }}
        </button>
      </div>
    </aside>

    <div class="stage">
      <p v-if="status?.state === 'running'" class="banner" role="status" aria-live="polite">
        正在把官方数据写入本地库
        <template v-if="status.phase"> · {{ status.phase }}</template>
        <template v-if="status.dirtyTotal"> · {{ status.dirtyDone }}/{{ status.dirtyTotal }} 本</template>
        <template v-if="status.elapsedSec"> · {{ status.elapsedSec }}s</template>
      </p>
      <p v-if="status?.lastError" class="error" role="alert">上次同步失败：{{ status.lastError }}</p>
      <main id="main" tabindex="-1">
        <RouterView v-slot="{ Component }">
          <Transition name="page" mode="out-in">
            <component :is="Component" />
          </Transition>
        </RouterView>
      </main>
      <footer class="colophon">{{ siteTitle }} · 日常只读本地库，点同步才会请求微信读书</footer>
    </div>

    <DockNav />
  </div>
</template>
