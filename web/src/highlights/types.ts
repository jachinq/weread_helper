export type HighlightDisplay = 'card' | 'poster' | 'reader' | 'polaroid' | 'share'

export const HIGHLIGHT_DISPLAYS: {
  id: HighlightDisplay
  label: string
  hint: string
}[] = [
  { id: 'card', label: '藏书票', hint: '纸票散落，点开放大' },
  { id: 'poster', label: '海报', hint: '封面作底，引文叠在画面上' },
  { id: 'reader', label: '阅读', hint: '窄栏正文，少装饰' },
  { id: 'polaroid', label: '拍立得', hint: '上图下文，硬边白框' },
  { id: 'share', label: '分享图', hint: '正方形画幅，便于截图' },
]

export function normalizeHighlightDisplay(s: string | undefined): HighlightDisplay {
  const id = (s || '').toLowerCase().trim()
  if (HIGHLIGHT_DISPLAYS.some((x) => x.id === id)) return id as HighlightDisplay
  return 'card'
}
