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

export default function StudentDashboard() {
  const [templates, setTemplates] = useState([])
  const [requests, setRequests] = useState([])
  const [session, setSession] = useState(null)
  const [toast, setToast] = useState(null)
  const [apiError, setApiError] = useState(null)
  const [modal, setModal] = useState(null)

  const [desc, setDesc] = useState('')

  async function load() {
    const [tRes, rRes, sRes] = await Promise.all([
      api('/api/templates'),
      api('/api/request/my'),
      api('/api/session/me'),
    ])
    if (tRes.ok) setTemplates(tRes.data || [])
    if (rRes.ok) setRequests(rRes.data || [])
    if (sRes.ok) setSession(sRes.data)
  }

  useEffect(() => {
    load()
  }, [])

  useEffect(() => {
    if (!session?.active) return undefined
    const ping = () => {
      void api('/api/session/ping', { method: 'POST', body: '{}' })
    }
    ping()
    const id = setInterval(ping, 2 * 60 * 1000)
    return () => clearInterval(id)
  }, [session?.active])

  function notify(text, ok = true) {
    setToast({ text, ok })
    setTimeout(() => setToast(null), 4000)
  }

  const activeVM = session && session.active
  const pendingCreate = requests.some((r) => r.type === 'create' && r.status === 'pending')
  /** Нельзя подать ещё одну заявку на создание: уже есть ВМ или ждётся решение по заявке. */
  const cannotRequestNewVM = !!activeVM || pendingCreate

  const blockParts = []
  if (activeVM) {
    blockParts.push(
      'Нельзя создать вторую виртуальную машину: у вас уже есть активная ВМ. Сначала запросите её удаление у администратора.',
    )
  }
  if (pendingCreate) {
    blockParts.push(
      'Заявка на создание ВМ уже отправлена и ожидает решения. Дождитесь одобрения или отмените заявку в таблице ниже.',
    )
  }
  const blockHint = blockParts.length > 0 ? blockParts.join(' ') : null

  async function submitCreate() {
    if (!desc.trim()) {
      notify('Укажите описание', false)
      return
    }
    const res = await api('/api/request/create', {
      method: 'POST',
      body: JSON.stringify({ template_id: modal.template_id, description: desc }),
    })
    if (!res.ok) {
      const msg = res.error || 'Не удалось создать заявку'
      setApiError(msg)
      notify(msg, false)
      return
    }
    setApiError(null)
    notify('Заявка отправлена')
    setModal(null)
    setDesc('')
    load()
  }

  async function submitDelete() {
    if (!desc.trim()) {
      notify('Укажите описание', false)
      return
    }
    const res = await api('/api/request/delete', {
      method: 'POST',
      body: JSON.stringify({ description: desc }),
    })
    if (!res.ok) {
      const msg = res.error || 'Ошибка'
      setApiError(msg)
      notify(msg, false)
      return
    }
    setApiError(null)
    notify('Заявка на удаление отправлена')
    setModal(null)
    setDesc('')
    load()
  }

  async function cancelRequest(id) {
    const res = await api('/api/request/cancel', {
      method: 'POST',
      body: JSON.stringify({ request_id: id }),
    })
    if (!res.ok) {
      notify(res.error || 'Ошибка', false)
      return
    }
    notify('Заявка отменена')
    setApiError(null)
    load()
  }

  return (
    <div>
      {toast && <div className={`notice ${toast.ok ? 'ok' : 'err'}`}>{toast.text}</div>}

      {blockHint && (
        <div className="notice warn" role="status">
          <strong>Новая ВМ сейчас недоступна.</strong> {blockHint}
        </div>
      )}

      {apiError && (
        <div className="notice err" role="alert">
          <strong>Ошибка.</strong> {apiError}
        </div>
      )}

      <div className="card">
        <h2 style={{ marginTop: 0 }}>Шаблоны</h2>
        <p style={{ color: 'var(--muted)', fontSize: '0.9rem' }}>
          Запуск ВМ только после одобрения заявки администратором. Одновременно допускается только одна активная ВМ.
        </p>
        <ul style={{ listStyle: 'none', padding: 0 }}>
          {templates.map((t) => (
            <li
              key={t.id}
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                gap: 12,
                padding: '0.5rem 0',
                borderBottom: '1px solid var(--border)',
              }}
            >
              <div>
                <strong>{t.name}</strong>
                <div style={{ color: 'var(--muted)', fontSize: '0.85rem' }}>{t.description}</div>
              </div>
              <button
                type="button"
                className={cannotRequestNewVM ? 'btn-disabled' : 'btn-primary'}
                disabled={cannotRequestNewVM}
                title={
                  cannotRequestNewVM
                    ? activeVM
                      ? 'Сначала удалите текущую ВМ через заявку'
                      : 'Уже есть ожидающая заявка на создание'
                    : 'Запросить виртуальную машину'
                }
                onClick={() => {
                  if (cannotRequestNewVM) return
                  setDesc('')
                  setModal({ kind: 'create', template_id: t.id })
                }}
              >
                Запросить ВМ
              </button>
            </li>
          ))}
        </ul>
      </div>

      {activeVM && (
        <div className="card">
          <h3 style={{ marginTop: 0 }}>Активная ВМ</h3>
          {session.container_running ? (
            <p>
              Ссылка:{' '}
              <a href={session.url} target="_blank" rel="noreferrer">
                {session.url}
              </a>
            </p>
          ) : (
            <p style={{ color: 'var(--muted)' }}>Контейнер остановлен или перезапускается. Обновите страницу позже.</p>
          )}
          <p>
            Пароль VNC: <code>{session.password || 'vncpassword'}</code>
          </p>
          <p style={{ color: 'var(--muted)', fontSize: '0.85rem' }}>
            Пока открыт этот кабинет, раз в ~2 минуты отправляется «присутствие». Если не заходить сюда дольше заданного времени
            (по умолчанию около часа, см. <code>SESSION_IDLE_MINUTES</code> на сервере), ВМ будет остановлена автоматически.
          </p>
          <p style={{ color: 'var(--muted)', fontSize: '0.85rem' }}>
            На рабочем столе noVNC должен быть ярлык <strong>Geany</strong> (если используется образ `make up`). Откройте его двойным щелчком.
          </p>
          <button type="button" className="btn-warn" onClick={() => { setDesc(''); setModal({ kind: 'delete' }) }}>
            Запросить удаление
          </button>
        </div>
      )}

      <div className="card">
        <h2 style={{ marginTop: 0 }}>Мои заявки</h2>
        <table className="table">
          <thead>
            <tr>
              <th>Шаблон</th>
              <th>Описание</th>
              <th>Тип</th>
              <th>Статус</th>
              <th>Комментарий</th>
              <th>Дата</th>
              <th>Ссылка</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {requests.map((r) => (
              <tr key={r.id}>
                <td>{r.template_name}</td>
                <td>{r.description}</td>
                <td>{typeLabel[r.type] || r.type}</td>
                <td>{statusLabel[r.status] || r.status}</td>
                <td>{r.admin_comment || '—'}</td>
                <td>{r.created_at}</td>
                <td>
                  {r.vm_url ? (
                    <a href={r.vm_url} target="_blank" rel="noreferrer">
                      Открыть
                    </a>
                  ) : (
                    '—'
                  )}
                </td>
                <td>
                  {r.status === 'pending' && (
                    <button type="button" className="btn-ghost" onClick={() => cancelRequest(r.id)}>
                      Отменить
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {modal && (
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
            <h3>{modal.kind === 'create' ? 'Заявка на создание ВМ' : 'Заявка на удаление ВМ'}</h3>
            <label>Описание / причина</label>
            <textarea rows={4} value={desc} onChange={(e) => setDesc(e.target.value)} style={{ marginBottom: 12 }} />
            <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
              <button type="button" className="btn-ghost" onClick={() => setModal(null)}>
                Закрыть
              </button>
              <button
                type="button"
                className="btn-primary"
                onClick={modal.kind === 'create' ? submitCreate : submitDelete}
              >
                Отправить
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
