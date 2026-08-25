<script setup lang="ts">
defineProps<{
  starred: boolean
  busy?: boolean
}>()

const emit = defineEmits<{
  toggle: []
}>()
</script>

<template>
  <button
    class="star-mark"
    type="button"
    :class="{ on: starred }"
    :aria-pressed="starred"
    :aria-label="starred ? '取消星标' : '收入金句'"
    :disabled="busy"
    @click.stop="emit('toggle')"
  >
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path
        d="M12 3.6 14.4 9l6 .5-4.6 3.9 1.5 5.8L12 16.7 6.7 19.2 8.2 13.4 3.6 9.5l6-.5z"
        :fill="starred ? 'currentColor' : 'none'"
        stroke="currentColor"
        stroke-width="1.6"
        stroke-linejoin="round"
      />
    </svg>
  </button>
</template>
