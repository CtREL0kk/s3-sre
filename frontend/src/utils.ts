export function formatBytes(n: number | null): string {
  if (n == null) return ''
  if (n === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.min(Math.floor(Math.log(n) / Math.log(1024)), units.length - 1)
  const val = n / Math.pow(1024, i)
  return `${val.toFixed(val >= 100 || i === 0 ? 0 : 1)} ${units[i]}`
}

export function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('ru-RU', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  })
}


export interface UrlState {
  folderId: string | null
  previewId: string | null
  graph: boolean
}

export function readUrlState(): UrlState {
  const sp = new URLSearchParams(window.location.search)
  return {
    folderId: sp.get('folder'),
    previewId: sp.get('preview'),
    graph: sp.get('graph') === '1',
  }
}

export function writeUrlState(s: UrlState) {
  const sp = new URLSearchParams()
  if (s.folderId) sp.set('folder', s.folderId)
  if (s.previewId) sp.set('preview', s.previewId)
  if (s.graph) sp.set('graph', '1')
  const qs = sp.toString()
  window.history.replaceState(null, '', qs ? `?${qs}` : window.location.pathname)
}