import React, { useState } from 'react'
import { api, setToken } from './api.js'
import StudentDashboard from './StudentDashboard.jsx'
import AdminPanel from './AdminPanel.jsx'

export default function App() {
  const [role, setRole] = useState(localStorage.getItem('role') || '')
  const [studentId, setStudentId] = useState('')
  const [password, setPassword] = useState('')
  const [msg, setMsg] = useState(null)

  async function login(e) {
    e.preventDefault()
    setMsg(null)
    const data = await api('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify({ student_id: studentId, password }),
    })
    if (!data.ok) {
      setMsg({ type: 'err', text: data.error || 'Ошибка входа' })
      return
    }
    setToken(data.data.token)
    localStorage.setItem('role', data.data.role)
    setRole(data.data.role)
  }

  async function logout() {
    try {
      await api('/api/auth/logout', { method: 'POST', body: '{}' })
    } catch {
      /* сеть / сервер — всё равно выходим из UI */
    } finally {
      setToken('')
      localStorage.removeItem('role')
      setRole('')
    }
  }

  if (!role) {
    return (
      <div style={{ maxWidth: 420, margin: '4rem auto', padding: '0 1rem' }}>
        <div className="card">
          <h1 style={{ marginTop: 0 }}>Виртуальная лаборатория</h1>
          <p style={{ color: 'var(--muted)', fontSize: '0.9rem' }}>
            Вход по номеру студенческого билета (идентификатор) и паролю. Демо: студент{' '}
            <code>student1</code> / <code>student</code>, админ <code>admin</code> / <code>admin</code>.
          </p>
          {msg && <div className={`notice ${msg.type}`}>{msg.text}</div>}
          <form onSubmit={login}>
            <label>Идентификатор</label>
            <input value={studentId} onChange={(e) => setStudentId(e.target.value)} style={{ marginBottom: 12 }} />
            <label>Пароль</label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              style={{ marginBottom: 12 }}
            />
            <button type="submit" className="btn-primary" style={{ width: '100%', padding: '0.75rem' }}>
              Войти
            </button>
          </form>
        </div>
      </div>
    )
  }

  return (
    <div style={{ padding: '1rem', maxWidth: 1100, margin: '0 auto' }}>
      <header style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <strong>Виртуальная лаборатория</strong>
        <button type="button" className="btn-ghost" onClick={() => void logout()}>
          Выйти
        </button>
      </header>
      {role === 'admin' ? <AdminPanel /> : <StudentDashboard />}
    </div>
  )
}
