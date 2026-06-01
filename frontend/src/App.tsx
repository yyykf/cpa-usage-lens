import { useState } from 'react'
import Login from './pages/Login'
import Dashboard from './pages/Dashboard'
import { isAuthed } from './lib/api'

export default function App() {
  const [authed, setAuthed] = useState<boolean>(isAuthed())

  return authed ? (
    <Dashboard onLogout={() => setAuthed(false)} />
  ) : (
    <Login onSuccess={() => setAuthed(true)} />
  )
}
