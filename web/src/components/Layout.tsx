import { Link, Outlet, useNavigate } from 'react-router-dom'
import { clearSession, getStoredUser } from '../lib/auth'

export function Layout() {
  const navigate = useNavigate()
  const user = getStoredUser()

  function logout() {
    clearSession()
    navigate('/login', { replace: true })
  }

  return (
    <div className="min-h-screen bg-slate-50">
      <header className="border-b border-slate-200 bg-white">
        <div className="mx-auto flex max-w-6xl flex-col gap-3 px-4 py-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:gap-6">
            <Link to="/tasks" className="text-lg font-semibold text-slate-900">
              TaskFlow
            </Link>
          <nav aria-label="主导航" className="flex flex-wrap gap-4 text-sm">
            <Link to="/tasks" className="text-slate-600 hover:text-slate-900">
              任务
            </Link>
            <Link to="/tasks/new" className="text-slate-600 hover:text-slate-900">
              新建
            </Link>
            {user?.role === 'owner' ? (
              <Link to="/users" className="text-slate-600 hover:text-slate-900">
                用户
              </Link>
            ) : null}
            <Link to="/me" className="text-slate-600 hover:text-slate-900">
              我的
            </Link>
          </nav>
          </div>
          <div className="flex flex-wrap items-center gap-3 text-sm">
            {user ? (
              <span className="text-slate-600">
                {user.name} ({user.role})
              </span>
            ) : null}
            <button
              type="button"
              onClick={logout}
              className="rounded-md border border-slate-300 px-3 py-1.5 hover:bg-slate-50"
            >
              退出
            </button>
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-6xl px-4 py-6">
        <Outlet />
      </main>
    </div>
  )
}
