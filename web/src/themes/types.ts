export type ColorScheme = 'light' | 'dark'
export type LayoutId = 'classic'

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
  layout: LayoutId
  hasLamp: boolean
  fonts: {
    display: string
    serif: string
    ui: string
  }
  preview: ThemePreview
}
