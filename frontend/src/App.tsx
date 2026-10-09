import { useEffect, useState } from 'react'
import { api, isAuthed, setTokens } from './api'
import type { User } from './types'
import Login from './components/Login'
import Explorer from './components/Explorer'

export default function App() {
  const [user, setUser] = useState<User | null>(null)
  const [ready, setReady] = useState(false)

  useEffect(() => {
    if (!isAuthed()) {
      setReady(true)
      return
    }
    api
      .me()
      .then(setUser)
      .catch(() => setTokens(null, null))
      .finally(() => setReady(true))
  }, [])

  if (!ready) {
    return <div className="grid h-full place-items-center text-slate-400">Загрузка…</div>
  }

  if (!user) {
    return <Login onAuthed={setUser} />
  }

  return (
    <Explorer
      user={user}
      onLogout={() => {
        api.logout().catch(() => {})
        setTokens(null, null)
        setUser(null)
      }}
    />
  )
}