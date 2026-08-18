<script setup lang="ts">
import { computed } from 'vue'
import ClassicShell from './themes/classic/Shell.vue'
import { activeTheme } from './themes/apply'

const shells = {
  classic: ClassicShell,
}

const Shell = computed(() => shells[activeTheme.value.layout] ?? ClassicShell)
</script>

<template>
  <component :is="Shell" />
</template>
