<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'

const route = useRoute()
const path = computed(() => route.path)
</script>

<template>
  <nav class="notes-subnav modes" aria-label="笔记分区">
    <RouterLink class="btn" to="/notes" :class="{ active: path === '/notes' }" :aria-current="path === '/notes' ? 'page' : undefined">
      书单
    </RouterLink>
    <RouterLink
      class="btn"
      to="/notes/search"
      :class="{ active: path === '/notes/search' }"
      :aria-current="path === '/notes/search' ? 'page' : undefined"
    >
      检索
    </RouterLink>
    <RouterLink
      class="btn"
      to="/notes/starred"
      :class="{ active: path === '/notes/starred' }"
      :aria-current="path === '/notes/starred' ? 'page' : undefined"
    >
      金句
    </RouterLink>
  </nav>
</template>
