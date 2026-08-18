import type { ColorScheme, ThemeId, ThemeMeta } from './types'

export const DEFAULT_THEME_ID: ThemeId = 'walnut'
export const DEFAULT_SCHEME: ColorScheme = 'dark'

export const THEMES: ThemeMeta[] = [
  {
    id: 'walnut',
    label: '胡桃灯影',
    kicker: 'Lamp-lit library',
    layout: 'classic',
    hasLamp: true,
    fonts: {
      display: '"Fraunces", "Noto Serif SC", serif',
      serif: '"Noto Serif SC", "Songti SC", serif',
      ui: '"Karla", "Helvetica Neue", sans-serif',
    },
    preview: {
      dark: ['#16110d', '#c45c26', '#e8b86d'],
      light: ['#f6edd9', '#c45c26', '#8a5a2b'],
    },
  },
  {
    id: 'celadon',
    label: '青瓷书斋',
    kicker: 'Celadon study',
    layout: 'classic',
    hasLamp: false,
    fonts: {
      display: '"Noto Serif SC", "Songti SC", serif',
      serif: '"Noto Serif SC", serif',
      ui: '"Karla", sans-serif',
    },
    preview: {
      dark: ['#121816', '#7ea392', '#c9dccf'],
      light: ['#e7eee8', '#3f6b5c', '#1f2e28'],
    },
  },
  {
    id: 'cinnabar',
    label: '朱砂线装',
    kicker: 'Bound in vermilion',
    layout: 'classic',
    hasLamp: false,
    fonts: {
      display: '"Noto Serif SC", serif',
      serif: '"Noto Serif SC", serif',
      ui: '"Karla", sans-serif',
    },
    preview: {
      dark: ['#1a100e', '#c43c2a', '#e8c9a8'],
      light: ['#f4eadc', '#b13222', '#2a1612'],
    },
  },
  {
    id: 'inknight',
    label: '夜航墨水',
    kicker: 'Night passage',
    layout: 'sidebar',
    hasLamp: true,
    fonts: {
      display: '"Source Serif 4", "Noto Serif SC", serif',
      serif: '"Source Serif 4", "Noto Serif SC", serif',
      ui: '"Karla", sans-serif',
    },
    preview: {
      dark: ['#0d1420', '#6ea0d4', '#d7e4f5'],
      light: ['#eef3f8', '#1e4f86', '#122033'],
    },
  },
  {
    id: 'moss',
    label: '茶室苔绿',
    kicker: 'Tea-room moss',
    layout: 'classic',
    hasLamp: false,
    fonts: {
      display: '"Noto Serif SC", serif',
      serif: '"Noto Serif SC", serif',
      ui: '"Karla", sans-serif',
    },
    preview: {
      dark: ['#141a12', '#8aa35a', '#d8c8a0'],
      light: ['#eef0e2', '#4d6a2c', '#1e2418'],
    },
  },
  {
    id: 'moonfoil',
    label: '月光银箔',
    kicker: 'Moonfoil',
    layout: 'sidebar',
    hasLamp: true,
    fonts: {
      display: '"Cormorant Garamond", "Noto Serif SC", serif',
      serif: '"Cormorant Garamond", "Noto Serif SC", serif',
      ui: '"Karla", sans-serif',
    },
    preview: {
      dark: ['#141518', '#c4b48a', '#e8e6e1'],
      light: ['#f3f1ec', '#8a7344', '#1c1d20'],
    },
  },
  {
    id: 'persimmon',
    label: '秋柳柿红',
    kicker: 'Persimmon autumn',
    layout: 'classic',
    hasLamp: false,
    fonts: {
      display: '"Fraunces", "Noto Serif SC", serif',
      serif: '"Noto Serif SC", serif',
      ui: '"Karla", sans-serif',
    },
    preview: {
      dark: ['#1a120e', '#e07a3a', '#f0d9a8'],
      light: ['#f7efe4', '#c45a22', '#2a1a12'],
    },
  },
  {
    id: 'letterpress',
    label: '铅字排印',
    kicker: 'Letterpress',
    layout: 'sidebar',
    hasLamp: false,
    fonts: {
      display: '"IBM Plex Mono", "Noto Serif SC", monospace',
      serif: '"Noto Serif SC", serif',
      ui: '"IBM Plex Mono", "Karla", monospace',
    },
    preview: {
      dark: ['#161616', '#d0d0d0', '#f2ead8'],
      light: ['#f4f1ea', '#222', '#111'],
    },
  },
]

const byId = new Map(THEMES.map((t) => [t.id, t]))

export function isThemeId(v: string): v is ThemeId {
  return byId.has(v as ThemeId)
}

export function isColorScheme(v: string): v is ColorScheme {
  return v === 'light' || v === 'dark'
}

export function getTheme(id: string | undefined | null): ThemeMeta {
  if (id && isThemeId(id)) return byId.get(id)!
  return byId.get(DEFAULT_THEME_ID)!
}

export function normalizeScheme(v: string | undefined | null): ColorScheme {
  return v === 'light' || v === 'dark' ? v : DEFAULT_SCHEME
}
