import { ChangeEvent, ReactNode, useEffect, useRef, useState } from 'react'
import {
  ArrowUp,
  Cloud,
  Download,
  File,
  FileArchive,
  FileImage,
  FileText,
  FileVideo,
  Folder,
  FolderPlus,
  Globe,
  Loader2,
  Lock,
  LogOut,
  Move,
  Network,
  Pencil,
  Trash2,
  Upload,
  UserRound,
  Users,
} from 'lucide-react'
import { api, downloadObject } from '../api'
import type { ObjectItem, User } from '../types'
import { formatBytes, formatDate, readUrlState, writeUrlState } from '../utils'
import { GrantsDialog, MoveDialog, PromptDialog } from './Dialogs'
import GraphDialog from './GraphDialog'
import PreviewDialog from './PreviewDialog'
import ProfileDialog from './ProfileDialog'

function fileIcon(item: ObjectItem) {
  const ct = item.content_type ?? ''
  if (ct.startsWith('image/')) return <FileImage className="h-5 w-5 text-emerald-400" />
  if (ct.startsWith('video/')) return <FileVideo className="h-5 w-5 text-purple-400" />
  if (ct.startsWith('text/') || ct.includes('pdf')) return <FileText className="h-5 w-5 text-sky-400" />
  if (['zip', 'gzip', 'x-7z'].some((s) => ct.includes(s))) return <FileArchive className="h-5 w-5 text-amber-400" />
  return <File className="h-5 w-5 text-slate-400" />
}

async function buildPath(folderId: string): Promise<ObjectItem[]> {
  const items: ObjectItem[] = []
  let cur: string | null = folderId
  while (cur) {
    const obj = await api.getObject(cur)
    items.unshift(obj)
    cur = obj.parent_id
  }
  return items
}

type Dialog =
  | { kind: 'newFolder' }
  | { kind: 'rename'; item: ObjectItem }
  | { kind: 'move'; item: ObjectItem }
  | { kind: 'grants'; item: ObjectItem }
  | { kind: 'preview'; item: ObjectItem }
  | { kind: 'graph' }
  | null

export default function Explorer({ user, onLogout }: { user: User; onLogout: () => void }) {
  const [path, setPath] = useState<ObjectItem[]>([])
  const [items, setItems] = useState<ObjectItem[]>([])
  const [loading, setLoading] = useState(true)
  const [booting, setBooting] = useState(true)
  const [error, setError] = useState('')
  const [dialog, setDialog] = useState<Dialog>(null)
  const [profileOpen, setProfileOpen] = useState(false)
  const [readOnly, setReadOnly] = useState(false)
  const fileInput = useRef<HTMLInputElement>(null)
  const skipUrlSync = useRef(true)

  const current = path[path.length - 1] ?? null

  useEffect(() => {
    const { folderId, previewId, graph } = readUrlState()
    ;(async () => {
      try {
        const cfg = await api.getConfig()
        setReadOnly(cfg.storage_mode === 'readonly')
        if (folderId) {
          setPath(await buildPath(folderId))
        }
        if (previewId) {
          const obj = await api.getObject(previewId)
          setDialog({ kind: 'preview', item: obj })
        } else if (graph) {
          setDialog({ kind: 'graph' })
        }
      } catch {
      } finally {
        setBooting(false)
      }
    })()
  }, [])

  useEffect(() => {
    if (skipUrlSync.current) {
      skipUrlSync.current = false
      return
    }
    writeUrlState({
      folderId: current?.id ?? null,
      previewId: dialog?.kind === 'preview' ? dialog.item.id : null,
      graph: dialog?.kind === 'graph',
    })
  }, [path, dialog])

  async function refresh() {
    setLoading(true)
    setError('')
    try {
      const data = current ? await api.listChildren(current.id) : await api.listRoot()
      setItems(data.items)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Ошибка загрузки')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (booting) return
    refresh()
  }, [current?.id, booting])

  function openFolder(item: ObjectItem) {
    setPath((p) => [...p, item])
  }

  async function doCreateFolder(name: string) {
    await api.createFolder(current?.id ?? null, name)
    await refresh()
  }

  async function doUpload(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    e.target.value = ''
    if (!file || !current) return
    try {
      setError('')
      await api.uploadFile(current.id, file)
      await refresh()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка загрузки')
    }
  }

  async function doRename(item: ObjectItem, name: string) {
    await api.update(item.id, { name })
    await refresh()
  }

  async function doDelete(item: ObjectItem) {
    if (!window.confirm(`Удалить «${item.name}»?`)) return
    try {
      await api.remove(item.id)
      await refresh()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка удаления')
    }
  }

  async function doToggleVisibility(item: ObjectItem) {
    try {
      await api.setVisibility(item.id, item.visibility === 'public' ? 'private' : 'public')
      await refresh()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка')
    }
  }

  async function doDownload(item: ObjectItem) {
    try {
      await downloadObject(item.id, item.name)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка скачивания')
    }
  }

  const toolbarBtn =
    'inline-flex items-center gap-1.5 rounded-lg border border-slate-700 bg-slate-800/60 px-3 py-2 text-sm font-medium text-slate-200 transition hover:bg-slate-700 hover:text-white'

  return (
    <div className="flex h-full flex-col">
      <header className="flex items-center justify-between border-b border-slate-800 bg-slate-900/50 px-5 py-3">
        <div className="flex items-center gap-2">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-br from-cyan-500 to-blue-600">
            <Cloud className="h-4.5 w-4.5 text-white" />
          </div>
          <span className="text-base font-bold text-slate-100">myS3</span>
        </div>
        <button
          onClick={() => setProfileOpen(true)}
          className="inline-flex items-center gap-1.5 rounded-lg px-3 py-2 text-sm text-slate-400 transition hover:bg-slate-800 hover:text-slate-100"
          title="Профиль"
        >
          <UserRound className="h-4 w-4" />
          <span className="hidden sm:inline">{user.username}</span>
        </button>
        <button
          onClick={onLogout}
          className="inline-flex items-center gap-1.5 rounded-lg px-3 py-2 text-sm text-slate-400 transition hover:bg-slate-800 hover:text-slate-100"
        >
          <LogOut className="h-4 w-4" /> Выйти
        </button>
      </header>

      <div className="flex items-center gap-1 overflow-x-auto border-b border-slate-800/60 px-5 py-2 text-sm">
        <button onClick={() => setPath([])} className="shrink-0 text-slate-300 hover:text-cyan-400">
          Мои файлы
        </button>
        {path.map((p, i) => (
          <span key={p.id} className="flex shrink-0 items-center gap-1">
            <span className="text-slate-600">/</span>
            <button
              onClick={() => setPath(path.slice(0, i + 1))}
              className="max-w-40 truncate text-slate-300 hover:text-cyan-400"
            >
              {p.name}
            </button>
          </span>
        ))}
      </div>

      <div className="flex flex-wrap items-center gap-2 px-5 py-3">
        {current && (
          <button className={toolbarBtn} onClick={() => setPath(path.slice(0, -1))} title="На уровень выше">
            <ArrowUp className="h-4 w-4" /> Наверх
          </button>
        )}
        {!readOnly && (
          <button className={toolbarBtn} onClick={() => setDialog({ kind: 'newFolder' })}>
            <FolderPlus className="h-4 w-4" /> Папка
          </button>
        )}
        {!readOnly && (
          <button className={toolbarBtn} onClick={() => fileInput.current?.click()}>
            <Upload className="h-4 w-4" /> Загрузить
          </button>
        )}
        <button className={toolbarBtn} onClick={() => setDialog({ kind: 'graph' })} title="Граф связей">
          <Network className="h-4 w-4" /> Граф
        </button>
        <input ref={fileInput} type="file" className="hidden" onChange={doUpload} />
        {readOnly && (
          <span className="ml-auto inline-flex items-center gap-1.5 rounded-full border border-amber-600/50 bg-amber-950/40 px-3 py-1 text-xs font-medium text-amber-300">
            <Lock className="h-3.5 w-3.5" /> READONLY
          </span>
        )}
        {error && <span className="ml-auto text-sm text-red-400">{error}</span>}
      </div>

      <div className="flex-1 overflow-y-auto px-5 pb-6">
        {loading ? (
          <div className="flex items-center justify-center gap-2 py-16 text-slate-400">
            <Loader2 className="h-5 w-5 animate-spin" /> Загрузка…
          </div>
        ) : items.length === 0 ? (
          <div className="py-16 text-center text-slate-500">
            <Folder className="mx-auto mb-3 h-10 w-10 text-slate-700" />
            Папка пуста. Создайте папку или загрузите файл.
          </div>
        ) : (
          <div className="overflow-hidden rounded-xl border border-slate-800 bg-slate-900/40">
            {items.map((item, i) => (
              <div
                key={item.id}
                className={
                  'group flex items-center gap-3 px-4 py-3 transition hover:bg-slate-800/40 ' +
                  (i !== items.length - 1 ? 'border-b border-slate-800/60' : '')
                }
              >
                <div
                  className="flex min-w-0 flex-1 cursor-pointer items-center gap-3"
                  onClick={() => (item.type === 'folder' ? openFolder(item) : setDialog({ kind: 'preview', item }))}
                >
                  {item.type === 'folder' ? (
                    <Folder className="h-5 w-5 shrink-0 text-amber-400" />
                  ) : (
                    fileIcon(item)
                  )}
                  <div className="min-w-0">
                    <div className="truncate text-sm font-medium text-slate-100">{item.name}</div>
                    <div className="text-xs text-slate-500">
                      {item.type === 'file' ? formatBytes(item.size_bytes) : 'папка'} · {formatDate(item.updated_at)}
                    </div>
                  </div>
                </div>

                {!readOnly && (
                  <button
                    title={item.visibility === 'public' ? 'Публичный — скрыть' : 'Приватный — сделать публичным'}
                    onClick={() => doToggleVisibility(item)}
                    className={
                      'flex h-8 w-8 shrink-0 items-center justify-center rounded-lg transition ' +
                      (item.visibility === 'public'
                        ? 'text-emerald-400 hover:bg-emerald-950/40'
                        : 'text-slate-500 hover:bg-slate-800')
                    }
                  >
                    {item.visibility === 'public' ? <Globe className="h-4 w-4" /> : <Lock className="h-4 w-4" />}
                  </button>
                )}

                <div className="flex shrink-0 items-center opacity-0 transition group-hover:opacity-100">
                  {item.type === 'file' && (
                    <IconBtn title="Скачать" onClick={() => doDownload(item)}>
                      <Download className="h-4 w-4" />
                    </IconBtn>
                  )}
                  {!readOnly && (
                    <IconBtn title="Переименовать" onClick={() => setDialog({ kind: 'rename', item })}>
                      <Pencil className="h-4 w-4" />
                    </IconBtn>
                  )}
                  {!readOnly && (
                    <IconBtn title="Переместить" onClick={() => setDialog({ kind: 'move', item })}>
                      <Move className="h-4 w-4" />
                    </IconBtn>
                  )}
                  {!readOnly && (
                    <IconBtn title="Права доступа" onClick={() => setDialog({ kind: 'grants', item })}>
                      <Users className="h-4 w-4" />
                    </IconBtn>
                  )}
                  {!readOnly && (
                    <IconBtn title="Удалить" danger onClick={() => doDelete(item)}>
                      <Trash2 className="h-4 w-4" />
                    </IconBtn>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {dialog?.kind === 'newFolder' && (
        <PromptDialog
          title="Новая папка"
          placeholder="Название папки"
          onSubmit={doCreateFolder}
          onClose={() => setDialog(null)}
        />
      )}
      {dialog?.kind === 'rename' && (
        <PromptDialog
          title="Переименовать"
          placeholder="Новое имя"
          initial={dialog.item.name}
          onSubmit={(name) => doRename(dialog.item, name)}
          onClose={() => setDialog(null)}
        />
      )}
      {dialog?.kind === 'move' && (
        <MoveDialog objectId={dialog.item.id} onClose={() => setDialog(null)} />
      )}
      {dialog?.kind === 'grants' && (
        <GrantsDialog object={dialog.item} onClose={() => setDialog(null)} />
      )}
      {dialog?.kind === 'preview' && (
        <PreviewDialog item={dialog.item} onClose={() => setDialog(null)} />
      )}
      {dialog?.kind === 'graph' && (
        <GraphDialog folderId={current?.id ?? null} onClose={() => setDialog(null)} />
      )}
      {profileOpen && <ProfileDialog user={user} onClose={() => setProfileOpen(false)} />}
    </div>
  )
}

function IconBtn({
  title,
  danger,
  onClick,
  children,
}: {
  title: string
  danger?: boolean
  onClick: () => void
  children: ReactNode
}) {
  return (
    <button
      title={title}
      onClick={onClick}
      className={
        'flex h-8 w-8 items-center justify-center rounded-lg text-slate-400 transition hover:bg-slate-700 ' +
        (danger ? 'hover:text-red-400' : 'hover:text-slate-100')
      }
    >
      {children}
    </button>
  )
}
