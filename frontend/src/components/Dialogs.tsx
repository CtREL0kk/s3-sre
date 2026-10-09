import { FormEvent, ReactNode, useEffect, useState } from 'react'
import { ChevronRight, Folder, Trash2, UserPlus } from 'lucide-react'
import { api } from '../api'
import type { Grant, ObjectItem, Permission } from '../types'

export function Modal({ children, onClose }: { children: ReactNode; onClose: () => void }) {
  return (
    <div className="fixed inset-0 z-50 grid place-items-center bg-black/60 p-4" onClick={onClose}>
      <div
        className="w-full max-w-md rounded-xl border border-slate-800 bg-slate-900 p-5 shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        {children}
      </div>
    </div>
  )
}

function FieldError({ message }: { message: string }) {
  if (!message) return null
  return (
    <div className="rounded-lg border border-red-900/50 bg-red-950/40 px-3 py-2 text-sm text-red-300">
      {message}
    </div>
  )
}

export function PromptDialog({
  title,
  placeholder,
  initial = '',
  onSubmit,
  onClose,
}: {
  title: string
  placeholder: string
  initial?: string
  onSubmit: (value: string) => void | Promise<void>
  onClose: () => void
}) {
  const [value, setValue] = useState(initial)
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')

  async function submit(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setErr('')
    try {
      await onSubmit(value.trim())
      onClose()
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Ошибка')
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal onClose={onClose}>
      <h3 className="mb-3 text-lg font-semibold text-slate-100">{title}</h3>
      <form onSubmit={submit} className="space-y-3">
        <input
          autoFocus
          value={value}
          onChange={(e) => setValue(e.target.value)}
          placeholder={placeholder}
          className="w-full rounded-lg border border-slate-700 bg-slate-800 px-3.5 py-2.5 text-sm text-slate-100 placeholder:text-slate-500 focus:border-cyan-500 focus:outline-none"
        />
        <FieldError message={err} />
        <div className="flex justify-end gap-2">
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg px-4 py-2 text-sm text-slate-300 hover:bg-slate-800"
          >
            Отмена
          </button>
          <button
            type="submit"
            disabled={busy || !value.trim()}
            className="rounded-lg bg-cyan-600 px-4 py-2 text-sm font-medium text-white hover:bg-cyan-500 disabled:opacity-50"
          >
            {busy ? '…' : 'Сохранить'}
          </button>
        </div>
      </form>
    </Modal>
  )
}

export function MoveDialog({ objectId, onClose }: { objectId: string; onClose: () => void }) {
  const [path, setPath] = useState<ObjectItem[]>([])
  const [items, setItems] = useState<ObjectItem[]>([])
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')
  const current = path[path.length - 1] ?? null

  useEffect(() => {
    const load = async () => {
      const data = current ? await api.listChildren(current.id) : await api.listRoot()
      setItems(data.items.filter((i) => i.type === 'folder'))
    }
    load().catch((e) => setErr(e instanceof Error ? e.message : 'Ошибка'))
  }, [current?.id])

  async function moveHere() {
    setBusy(true)
    setErr('')
    try {
      await api.update(objectId, { parent_id: current!.id })
      onClose()
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Ошибка')
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal onClose={onClose}>
      <h3 className="mb-3 text-lg font-semibold text-slate-100">Переместить</h3>

      <div className="mb-3 flex flex-wrap items-center gap-1 text-sm text-slate-300">
        <button className="hover:text-cyan-400" onClick={() => setPath([])}>
          Корень
        </button>
        {path.map((p, i) => (
          <span key={p.id} className="flex items-center gap-1">
            <ChevronRight className="h-3.5 w-3.5 text-slate-600" />
            <button className="hover:text-cyan-400" onClick={() => setPath(path.slice(0, i + 1))}>
              {p.name}
            </button>
          </span>
        ))}
      </div>

      {current && (
        <button
          onClick={() => setPath(path.slice(0, -1))}
          className="mb-2 text-sm text-slate-400 hover:text-slate-200"
        >
          ← На уровень выше
        </button>
      )}

      <div className="mb-4 max-h-60 space-y-1 overflow-y-auto rounded-lg border border-slate-800 bg-slate-950/40 p-2">
        {items.length === 0 && <div className="p-3 text-sm text-slate-500">Нет вложенных папок</div>}
        {items.map((f) => (
          <button
            key={f.id}
            onClick={() => setPath([...path, f])}
            className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm text-slate-200 hover:bg-slate-800"
          >
            <Folder className="h-4 w-4 text-amber-400" />
            {f.name}
          </button>
        ))}
      </div>

      <FieldError message={err} />
      <div className="flex justify-end gap-2">
        <button onClick={onClose} className="rounded-lg px-4 py-2 text-sm text-slate-300 hover:bg-slate-800">
          Отмена
        </button>
        {current && (
          <button
            onClick={moveHere}
            disabled={busy}
            className="rounded-lg bg-cyan-600 px-4 py-2 text-sm font-medium text-white hover:bg-cyan-500 disabled:opacity-50"
          >
            {busy ? '…' : `Переместить в «${current.name}»`}
          </button>
        )}
      </div>
    </Modal>
  )
}

export function GrantsDialog({ object, onClose }: { object: ObjectItem; onClose: () => void }) {
  const [grants, setGrants] = useState<Grant[]>([])
  const [grantee, setGrantee] = useState('')
  const [perm, setPerm] = useState<Permission>('read')
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')

  useEffect(() => {
    api.listGrants(object.id).then((d) => setGrants(d.items)).catch(() => setGrants([]))
  }, [object.id])

  async function add() {
    setBusy(true)
    setErr('')
    try {
      await api.setGrant(object.id, grantee.trim(), perm)
      setGrantee('')
      setGrants((await api.listGrants(object.id)).items)
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Ошибка')
    } finally {
      setBusy(false)
    }
  }

  async function remove(granteeId: string) {
    await api.deleteGrant(object.id, granteeId)
    setGrants((await api.listGrants(object.id)).items)
  }

  return (
    <Modal onClose={onClose}>
      <h3 className="mb-1 text-lg font-semibold text-slate-100">Права доступа</h3>
      <p className="mb-4 text-sm text-slate-400">
        «{object.name}» — наследующийся на всё поддерево
      </p>

      <div className="mb-4 space-y-2">
        {grants.length === 0 && <div className="text-sm text-slate-500">Права никому не выданы</div>}
        {grants.map((g) => (
          <div
            key={g.grantee_id}
            className="flex items-center justify-between rounded-lg border border-slate-800 bg-slate-950/40 px-3 py-2"
          >
            <div>
              <div className="font-mono text-xs text-slate-300">{g.grantee_id}</div>
              <div className="text-xs text-slate-500">{g.permission === 'read' ? 'чтение' : 'чтение и запись'}</div>
            </div>
            <button onClick={() => remove(g.grantee_id)} className="text-slate-500 hover:text-red-400" title="Отозвать">
              <Trash2 className="h-4 w-4" />
            </button>
          </div>
        ))}
      </div>

      <div className="space-y-2 rounded-lg border border-slate-800 bg-slate-950/40 p-3">
        <label className="text-xs text-slate-400">ID пользователя (uuid)</label>
        <input
          value={grantee}
          onChange={(e) => setGrantee(e.target.value)}
          placeholder="00000000-0000-0000-0000-000000000000"
          className="w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 font-mono text-xs text-slate-100 focus:border-cyan-500 focus:outline-none"
        />
        <div className="flex items-center gap-2">
          <select
            value={perm}
            onChange={(e) => setPerm(e.target.value as Permission)}
            className="rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-sm text-slate-100"
          >
            <option value="read">Чтение</option>
            <option value="write">Запись</option>
          </select>
          <button
            onClick={add}
            disabled={busy || !grantee.trim()}
            className="flex flex-1 items-center justify-center gap-1.5 rounded-lg bg-cyan-600 px-3 py-2 text-sm text-white hover:bg-cyan-500 disabled:opacity-50"
          >
            <UserPlus className="h-4 w-4" /> Выдать
          </button>
        </div>
        <FieldError message={err} />
      </div>

      <div className="mt-4 flex justify-end">
        <button onClick={onClose} className="rounded-lg px-4 py-2 text-sm text-slate-300 hover:bg-slate-800">
          Закрыть
        </button>
      </div>
    </Modal>
  )
}