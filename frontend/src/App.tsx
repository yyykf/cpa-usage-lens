import { useState } from 'react'
import Login from './pages/Login'
import Dashboard from './pages/Dashboard'
import { isAuthed } from './lib/api'

export default function App() {
  const previewMode =
    import.meta.env.DEV &&
    typeof window !== 'undefined' &&
    new URLSearchParams(window.location.search).get('preview') === '1'
  const [authed, setAuthed] = useState<boolean>(isAuthed())

  return authed || previewMode ? (
    <Dashboard
      onLogout={() => {
        setAuthed(false)
        if (previewMode) window.location.href = '/'
      }}
    />
  ) : (
    <Login onSuccess={() => setAuthed(true)} />
  )
}
