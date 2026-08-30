export type HighlightDisplay = 'card' | 'poster' | 'reader' | 'polaroid' | 'share'

export const HIGHLIGHT_DISPLAYS: {
  id: HighlightDisplay
  label: string
  hint: string
}[] = [
  { id: 'card', label: '藏书票', hint: '纸票散落，右下竖封可辨' },
  { id: 'poster', label: '海报', hint: '列表铺底，点开完整竖封' },
  { id: 'reader', label: '阅读', hint: '夜读墨底，金线栏标' },
  { id: 'polaroid', label: '拍立得', hint: '2:3 相纸，封面完整' },
  { id: 'share', label: '分享图', hint: '左封右文，点开看全文' },
]

export function normalizeHighlightDisplay(s: string | undefined): HighlightDisplay {
  const id = (s || '').toLowerCase().trim()
  if (HIGHLIGHT_DISPLAYS.some((x) => x.id === id)) return id as HighlightDisplay
  return 'card'
}
