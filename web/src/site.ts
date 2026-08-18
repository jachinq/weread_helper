import { ref } from 'vue'

export const siteTitle = ref('纸间笔记')

export function applySiteTitle(title: string) {
  const t = title.trim()
  siteTitle.value = t || '纸间笔记'
}
