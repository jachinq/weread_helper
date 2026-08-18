import { ref } from 'vue'

export const siteBrand = ref('纸间笔记')

export function applySiteTitle(page?: string) {
  const brand = siteBrand.value || '纸间笔记'
  document.title = page ? `${page} · ${brand}` : `${brand} · 微信读书助手`
}
