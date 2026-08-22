function isInline(url: string) {
  return url.startsWith('data:') || url.startsWith('blob:')
}

function parseMaybe(url: string) {
  try {
    return new URL(url, typeof location !== 'undefined' ? location.origin : 'http://127.0.0.1')
  } catch {
    return null
  }
}

function isCoverProxyPath(pathname: string) {
  return pathname === '/api/covers' || pathname.endsWith('/api/covers')
}

/** 同源封面代理地址保持相对路径，避免把 localhost 再套进 url= */
export function proxiedCover(url: string | undefined | null) {
  const u = url?.trim() || ''
  if (!u || isInline(u)) return u

  const parsed = parseMaybe(u)
  if (parsed && isCoverProxyPath(parsed.pathname)) {
    return `/api/covers${parsed.search}`
  }
  if (u.startsWith('/api/covers')) return u
  if (!/^https?:\/\//i.test(u)) return u

  const host = parsed?.hostname.toLowerCase() || ''
  if (host === 'localhost' || host === '127.0.0.1' || host === '::1') return u

  return `/api/covers?url=${encodeURIComponent(u)}`
}
