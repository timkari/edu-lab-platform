import React, { useEffect, useState } from 'react'
import { api } from './api.js'

const statusLabel = {
  pending: 'Ожидает',
  approved: 'Одобрена',
  rejected: 'Отклонена',
  cancelled: 'Отменена',
}

const typeLabel = {
  create: 'Создание',
  delete: 'Удаление',
}

export default function AdminPanel() {
  const [tab, setTab] = useState('requests')
  const [requests, setRequests] = useState([])
  const [students, setStudents] = useState([])
  const [filterStatus, setFilterStatus] = useState('')
  const [filterType, setFilterType] = useState('')
  const [toast, setToast] = useState(null)
  const [rejectModal, setRejectModal] = useState(null)
  const [rejectComment, setRejectComment] = useState('')

  function notify(text, ok = true) {
    setToast({ text, ok })
    setTimeout(() => setToast(null), 4000)
  }

  async function loadRequests() {
    const qs = new URLSearchParams()
    if (filterStatus) qs.set('status', filterStatus)
    if (filterType) qs.set('type', filterType)
    const res = await api('/api/request/all?' + qs.toString())
    if (res.ok) setRequests(res.data || [])
    else notify(res.error || 'Не удалось загрузить заявки', false)
  }

  async function loadStudents() {
    const res = await api('/api/list')
    if (res.ok) setStudents(res.data || [])
  }

  useEffect(() => {
    if (tab === 'requests') loadRequests()
    else loadStudents()
  }, [tab, filterStatus, filterType])

  async function approve(id) {
    const res = await api('/api/request/approve', {
      method: 'POST',
      body: JSON.stringify({ request_id: id }),
    })
    if (!res.ok) {
      notify(res.error || 'Ошибка', false)
      return
    }
    notify(res.data && res.data.url ? `ВМ запущена: ${res.data.url}` : 'Заявка обработана')
    loadRequests()
  }

  async function rejectSubmit() {
    if (!rejectModal) return
    const res = await api('/api/request/reject', {
      method: 'POST',
      body: JSON.stringify({ request_id: rejectModal, comment: rejectComment }),
    })
    if (!res.ok) {
      notify(res.error || 'Ошибка', false)
      return
    }
    notify('Заявка отклонена')
    setRejectModal(null)
    setRejectComment('')
    loadRequests()
  }

  return (
    <div>
      {toast && <div className={`notice ${toast.ok ? 'ok' : 'err'}`}>{toast.text}</div>}

      <div style={{ display: 'flex', gap: 8, marginBottom: 16 }}>
        <button
          type="button"
          className={tab === 'requests' ? 'btn-primary' : 'btn-ghost'}
          onClick={() => setTab('requests')}
        >
          Заявки
        </button>
        <button
          type="button"
          className={tab === 'students' ? 'btn-primary' : 'btn-ghost'}
          onClick={() => setTab('students')}
        >
          Студенты (диски)
        </button>
      </div>

      {tab === 'requests' && (
        <div className="card">
          <h2 style={{ marginTop: 0 }}>Заявки</h2>
          <div style={{ display: 'flex', gap: 12, marginBottom: 12, flexWrap: 'wrap' }}>
            <div>
              <label>Статус</label>
              <select value={filterStatus} onChange={(e) => setFilterStatus(e.target.value)}>
                <option value="">Все</option>
                <option value="pending">Ожидает</option>
                <option value="approved">Одобрена</option>
                <option value="rejected">Отклонена</option>
                <option value="cancelled">Отменена</option>
              </select>
            </div>
            <div>
              <label>Тип</label>
              <select value={filterType} onChange={(e) => setFilterType(e.target.value)}>
                <option value="">Все</option>
                <option value="create">Создание</option>
                <option value="delete">Удаление</option>
              </select>
            </div>
          </div>
          <table className="table">
            <thead>
              <tr>
                <th>ID</th>
                <th>Студент</th>
                <th>Шаблон</th>
                <th>Описание</th>
                <th>Тип</th>
                <th>Статус</th>
                <th>Дата</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {requests.map((r) => (
                <tr key={r.id}>
                  <td>{r.id}</td>
                  <td>{r.student_id}</td>
                  <td>{r.template_name}</td>
                  <td>{r.description}</td>
                  <td>{typeLabel[r.type] || r.type}</td>
                  <td>{statusLabel[r.status] || r.status}</td>
                  <td>{r.created_at}</td>
                  <td style={{ whiteSpace: 'nowrap' }}>
                    {r.status === 'pending' && (
                      <>
                        <button type="button" className="btn-primary" style={{ marginRight: 6 }} onClick={() => approve(r.id)}>
                          Одобрить
                        </button>
                        <button type="button" className="btn-ghost" onClick={() => { setRejectComment(''); setRejectModal(r.id) }}>
                          Отклонить
                        </button>
                      </>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {tab === 'students' && (
        <div className="card">
          <h2 style={{ marginTop: 0 }}>Каталоги на диске</h2>
          <table className="table">
            <thead>
              <tr>
                <th>ID</th>
                <th>Запущена</th>
                <th>Бэкапы</th>
                <th>Папка</th>
              </tr>
            </thead>
            <tbody>
              {students.map((s) => (
                <tr key={s.id}>
                  <td>{s.id}</td>
                  <td>{s.running ? 'Да' : 'Нет'}</td>
                  <td>{(s.backups || []).join(', ') || '—'}</td>
                  <td style={{ wordBreak: 'break-all' }}>{s.work_dir}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {rejectModal && (
        <div
          style={{
            position: 'fixed',
            inset: 0,
            background: 'rgba(0,0,0,0.6)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            padding: 16,
            zIndex: 50,
          }}
        >
          <div className="card" style={{ maxWidth: 480, width: '100%' }}>
            <h3>Отклонить заявку</h3>
            <label>Комментарий для студента</label>
            <textarea rows={3} value={rejectComment} onChange={(e) => setRejectComment(e.target.value)} style={{ marginBottom: 12 }} />
            <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
              <button type="button" className="btn-ghost" onClick={() => setRejectModal(null)}>
                Закрыть
              </button>
              <button type="button" className="btn-warn" onClick={rejectSubmit}>
                Отклонить
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
