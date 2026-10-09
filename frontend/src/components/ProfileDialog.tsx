import { useState } from 'react'
import { Check, Copy, Mail, User as UserIcon } from 'lucide-react'
import type { User } from '../types'
import { Modal } from './Dialogs'

export default function ProfileDialog({ user, onClose }: { user: User; onClose: () => void }) {
  const [copied, setCopied] = useState(false)

  async function copyId() {
    try {
      await navigator.clipboard.writeText(user.id)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
    }
  }

  const row = 'flex items-center gap-3 rounded-lg border border-slate-800 bg-slate-950/40 px-3 py-2.5'

  return (
    <Modal onClose={onClose}>
      <h3 className="mb-4 text-lg font-semibold text-slate-100">Профиль</h3>

      <div className="space-y-2">
        <div className={row}>
          <UserIcon className="h-4 w-4 shrink-0 text-cyan-400" />
          <div>
            <div className="text-xs text-slate-500">Имя пользователя</div>
            <div className="text-sm font-medium text-slate-100">{user.username}</div>
          </div>
        </div>

        <div className={row}>
          <Mail className="h-4 w-4 shrink-0 text-cyan-400" />
          <div>
            <div className="text-xs text-slate-500">Email</div>
            <div className="text-sm font-medium text-slate-100">{user.email}</div>
          </div>
        </div>

        <div className={row}>
          <div className="min-w-0">
            <div className="text-xs text-slate-500">ID (uuid)</div>
            <div className="truncate font-mono text-sm text-slate-100">{user.id}</div>
          </div>
          <button
            onClick={copyId}
            title="Скопировать ID"
            className="ml-auto flex h-8 w-8 shrink-0 items-center justify-center rounded-lg text-slate-400 hover:bg-slate-800 hover:text-slate-100"
          >
            {copied ? <Check className="h-4 w-4 text-emerald-400" /> : <Copy className="h-4 w-4" />}
          </button>
        </div>
      </div>

      <p className="mt-3 text-xs text-slate-500">
        Этот ID нужен, чтобы выдать вам права доступа на объекты других пользователей.
      </p>

      <div className="mt-4 flex justify-end">
        <button onClick={onClose} className="rounded-lg px-4 py-2 text-sm text-slate-300 hover:bg-slate-800">
          Закрыть
        </button>
      </div>
    </Modal>
  )
}