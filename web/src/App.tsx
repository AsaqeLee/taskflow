import { useEffect } from 'react'
import { Navigate, Route, Routes, useNavigate } from 'react-router-dom'
import { setSessionExpiredHandler } from './lib/session'
import { Layout } from './components/Layout'
import { isLoggedIn } from './lib/auth'
import { LoginPage } from './pages/LoginPage'
import { MePage } from './pages/MePage'
import { TaskDetailPage } from './pages/TaskDetailPage'
import { TaskListPage } from './pages/TaskListPage'
import { TaskNewPage } from './pages/TaskNewPage'

function RequireAuth({ children }: { children: React.ReactNode }) {
  if (!isLoggedIn()) {
    return <Navigate to="/login" replace />
  }
  return <>{children}</>
}

function SessionExpiryRedirect() {
  const navigate = useNavigate()

  useEffect(() => {
    setSessionExpiredHandler(() => {
      navigate('/login', { replace: true })
    })
  }, [navigate])

  return null
}

export default function App() {
  return (
    <>
      <SessionExpiryRedirect />
      <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route
        element={
          <RequireAuth>
            <Layout />
          </RequireAuth>
        }
      >
        <Route path="/" element={<Navigate to="/tasks" replace />} />
        <Route path="/tasks" element={<TaskListPage />} />
        <Route path="/tasks/new" element={<TaskNewPage />} />
        <Route path="/tasks/:id" element={<TaskDetailPage />} />
        <Route path="/me" element={<MePage />} />
      </Route>
      <Route path="*" element={<Navigate to="/tasks" replace />} />
      </Routes>
    </>
  )
}