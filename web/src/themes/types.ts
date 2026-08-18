export type ColorScheme = 'light' | 'dark'
export type LayoutId = 'classic' | 'sidebar'

export type ThemeId =
  | 'walnut'
  | 'celadon'
  | 'cinnabar'
  | 'inknight'
  | 'moss'
  | 'moonfoil'
  | 'persimmon'
  | 'letterpress'

export type ThemePreview = {
  light: [string, string, string]
  dark: [string, string, string]
}

export type ThemeMeta = {
  id: ThemeId
  label: string
  kicker: string
  /** L3 骨架 id；换导航结构时新增 layout 与 Shell */
  layout: LayoutId
  hasLamp: boolean
  fonts: {
    display: string
    serif: string
    ui: string
  }
  preview: ThemePreview
}
