import { domToBlob } from 'modern-screenshot'

function pad(n: number) {
  return String(n).padStart(2, '0')
}

function stampNow() {
  const d = new Date()
  return `${d.getFullYear()}${pad(d.getMonth() + 1)}${pad(d.getDate())}-${pad(d.getHours())}${pad(d.getMinutes())}${pad(d.getSeconds())}`
}

export function highlightExportFilename(title: string | undefined) {
  const raw = (title?.trim() || '未命名').replace(/[\\/:*?"<>|]+/g, '').replace(/\s+/g, ' ').trim()
  const safe = (raw || '未命名').slice(0, 40)
  return `${stampNow()}_${safe}.png`
}

async function waitForImages(root: HTMLElement) {
  const imgs = [...root.querySelectorAll('img')]
  await Promise.all(imgs.map((img) => img.decode().catch(() => undefined)))
}

export async function downloadHighlightCard(root: HTMLElement, filename: string) {
  const card = (root.firstElementChild as HTMLElement | null) ?? root
  card.classList.add('is-exporting')
  await waitForImages(card)
  await new Promise<void>((resolve) => requestAnimationFrame(() => resolve()))
  try {
    const blob = await domToBlob(card, {
      type: 'image/png',
      scale: 2,
      backgroundColor: null,
      style: { maxHeight: 'none', overflow: 'visible' },
    })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    a.rel = 'noopener'
    document.body.appendChild(a)
    a.click()
    a.remove()
    URL.revokeObjectURL(url)
  } finally {
    card.classList.remove('is-exporting')
  }
}
