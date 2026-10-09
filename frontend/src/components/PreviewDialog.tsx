import { useEffect, useState } from 'react'
import { AlertTriangle, Download, FileX2, Loader2, X } from 'lucide-react'
import { downloadObject, getContentBlob } from '../api'
import type { ObjectItem } from '../types'
import { formatBytes } from '../utils'
import { Modal } from './Dialogs'

const MAX_PREVIEW_SIZE = 5 * 1024 * 1024

type State = 'loading' | 'ready' | 'too-large' | 'unsupported' | 'error'

export default function PreviewDialog({ item, onClose }: { item: ObjectItem; onClose: () => void }) {
  const [state, setState] = useState<State>('loading')
  const [content, setContent] = useState('')
  const [blobUrl, setBlobUrl] = useState('')

  const contentType = item.content_type ?? ''

  useEffect(() => {
    let cancelled = false
    let url: string | null = null

    const isText = contentType.startsWith('text/') || contentType.includes('json')
    const isImage = contentType.startsWith('image/')

    if (!isText && !isImage) {
      setState('unsupported')
      return
    }
    if (item.size_bytes != null && item.size_bytes > MAX_PREVIEW_SIZE) {
      setState('too-large')
      return
    }

    getContentBlob(item.id)
      .then((blob) => {
        if (cancelled) return
        if (isImage) {
          url = URL.createObjectURL(blob)
          setBlobUrl(url)
          setState('ready')
        } else {
          blob.text().then((t) => {
            if (!cancelled) {
              setContent(t)
              setState('ready')
            }
          })
        }
      })
      .catch(() => {
        if (!cancelled) setState('error')
      })

    return () => {
      cancelled = true
      if (url) URL.revokeObjectURL(url)
    }
  }, [item.id])

  return (
    <Modal onClose={onClose}>
      <div className="mb-3 flex items-center justify-between gap-2">
        <h3 className="truncate text-lg font-semibold text-slate-100">{item.name}</h3>
        <div className="flex items-center gap-1">
          <button
            onClick={() => downloadObject(item.id, item.name)}
            className="flex h-8 w-8 items-center justify-center rounded-lg text-slate-400 hover:bg-slate-800 hover:text-slate-100"
            title="Скачать"
          >
            <Download className="h-4 w-4" />
          </button>
          <button
            onClick={onClose}
            className="flex h-8 w-8 items-center justify-center rounded-lg text-slate-400 hover:bg-slate-800 hover:text-slate-100"
            title="Закрыть"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
      </div>

      <div className="flex max-h-[70vh] min-h-[12rem] flex-col overflow-hidden rounded-lg border border-slate-800 bg-slate-950/40">
        {state === 'loading' && (
          <div className="flex flex-1 items-center justify-center gap-2 p-6 text-slate-400">
            <Loader2 className="h-5 w-5 animate-spin" /> Загрузка…
          </div>
        )}

        {state === 'ready' && blobUrl && (
          <div className="flex flex-1 items-center justify-center overflow-auto bg-slate-950 p-4">
            <img src={blobUrl} alt={item.name} className="max-h-[60vh] max-w-full object-contain" />
          </div>
        )}

        {state === 'ready' && !blobUrl && (
          <pre className="flex-1 overflow-auto whitespace-pre-wrap break-words p-4 font-mono text-sm text-slate-200">
            {content}
          </pre>
        )}

        {state === 'too-large' && (
          <div className="flex flex-1 flex-col items-center justify-center gap-3 p-6 text-center text-slate-300">
            <AlertTriangle className="h-8 w-8 text-amber-400" />
            <p>
              Файл больше {formatBytes(MAX_PREVIEW_SIZE)} — предпросмотр ограничен.
              <br />
              Скачайте файл для просмотра.
            </p>
          </div>
        )}

        {state === 'unsupported' && (
          <div className="flex flex-1 flex-col items-center justify-center gap-3 p-6 text-center text-slate-300">
            <FileX2 className="h-8 w-8 text-slate-500" />
            <p>Предпросмотр недоступен для этого типа файла.</p>
          </div>
        )}

        {state === 'error' && (
          <div className="flex flex-1 items-center justify-center p-6 text-red-300">
            Не удалось загрузить содержимое файла.
          </div>
        )}
      </div>
    </Modal>
  )
}