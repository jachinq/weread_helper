import { computed, ref } from 'vue'
import { DEFAULT_SCHEME, DEFAULT_THEME_ID, getTheme, normalizeScheme } from './registry'
import type { ColorScheme, ThemeMeta } from './types'

const STORAGE_KEY = 'weread.appearance'

export const activeTheme = ref<ThemeMeta>(getTheme(DEFAULT_THEME_ID))
export const activeScheme = ref<ColorScheme>(DEFAULT_SCHEME)

export const themeKicker = computed(() => activeTheme.value.kicker)
export const themeHasLamp = computed(() => activeTheme.value.hasLamp)

type StoredAppearance = {
  theme?: string
  scheme?: string
}

export function readStoredAppearance(): StoredAppearance {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return {}
    const parsed = JSON.parse(raw) as StoredAppearance
    return parsed && typeof parsed === 'object' ? parsed : {}
  } catch {
    return {}
  }
}

function writeStoredAppearance(themeId: string, scheme: ColorScheme) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ theme: themeId, scheme }))
  } catch {
    /* ignore quota / private mode */
  }
}

export function applyTheme(themeId?: string | null, scheme?: string | null) {
  const stored = readStoredAppearance()
  const theme = getTheme(themeId ?? stored.theme)
  const s = normalizeScheme(scheme ?? stored.scheme)
  activeTheme.value = theme
  activeScheme.value = s
  const el = document.documentElement
  el.dataset.theme = theme.id
  el.dataset.scheme = s
  el.dataset.layout = theme.layout
  el.dataset.lamp = theme.hasLamp ? '1' : '0'
  el.style.colorScheme = s
  writeStoredAppearance(theme.id, s)
}

export function bootTheme() {
  const stored = readStoredAppearance()
  applyTheme(stored.theme, stored.scheme)
}

export function applyThemeFromSettings(s: { theme?: string; colorScheme?: string }) {
  applyTheme(s.theme, s.colorScheme)
}
