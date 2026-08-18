<script setup lang="ts">
import { computed } from 'vue'
import type { Component } from 'vue'
import ClassicShell from './themes/classic/Shell.vue'
import SidebarShell from './themes/sidebar/Shell.vue'
import { activeTheme } from './themes/apply'
import type { LayoutId } from './themes/types'

const shells: Record<LayoutId, Component> = {
  classic: ClassicShell,
  sidebar: SidebarShell,
}

const Shell = computed(() => shells[activeTheme.value.layout] ?? ClassicShell)
</script>

<template>
  <component :is="Shell" />
</template>
