import { useCallback, useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { EmptyState } from '../components/EmptyState'
import { ErrorAlert } from '../components/ErrorAlert'
import { LoadingState } from '../components/LoadingState'
import { statusLabel } from '../domain/taskWorkflow'
import { useAsyncData } from '../hooks/useAsyncData'
import { fetchTasks } from '../lib/apiClient'
import type { TaskStatus } from '../types/api'

const PAGE_SIZE = 20
const statusOptions: TaskStatus[] = [
  'open',
  'assigned',
  'in_progress',
  'submitted',
  'approved',
  'completed',
  'cancelled',
]

export function TaskListPage() {
  const [queryInput, setQueryInput] = useState('')
  const [statusInput, setStatusInput] = useState<TaskStatus | ''>('')
  const [filters, setFilters] = useState<{
    q: string
    status: TaskStatus | ''
    page: number
  }>({
    q: '',
    status: '',
    page: 1,
  })

  const loadTasks = useCallback(
    () =>
      fetchTasks({
        q: filters.q || undefined,
        status: filters.status || undefined,
        page: filters.page,
        page_size: PAGE_SIZE,
      }),
    [filters],
  )

  const tasksState = useAsyncData(
    `tasks:${filters.q}:${filters.status || 'all'}:${filters.page}:${PAGE_SIZE}`,
    loadTasks,
  )

  function handleFilterSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()

    const nextQuery = queryInput.trim()
    if (filters.q === nextQuery && filters.status === statusInput && filters.page === 1) {
      tasksState.reload()
      return
    }

    setFilters({
      q: nextQuery,
      status: statusInput,
      page: 1,
    })
  }

  function handleClearFilters() {
    setQueryInput('')
    setStatusInput('')
    setFilters({
      q: '',
      status: '',
      page: 1,
    })
  }

  function changePage(page: number) {
    setFilters((current) => ({
      ...current,
      page,
    }))
  }

  if (tasksState.status === 'idle' || tasksState.status === 'loading') {
    return <LoadingState />
  }

  const list = tasksState.data ?? {
    tasks: [],
    total: 0,
    page: 1,
    page_size: PAGE_SIZE,
  }
  const tasks = list.tasks
  const error = tasksState.error
  const totalPages = Math.max(1, Math.ceil(list.total / list.page_size))
  const canGoPrev = list.page > 1
  const canGoNext = list.page < totalPages

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">任务列表</h1>
        <Link
          to="/tasks/new"
          className="rounded-md bg-blue-600 px-4 py-2 text-sm text-white"
        >
          新建任务
        </Link>
      </div>

      <ErrorAlert error={error} />

      <form
        className="grid gap-4 rounded-xl border border-slate-200 bg-white p-4 md:grid-cols-[minmax(0,2fr)_minmax(0,1fr)_auto]"
        onSubmit={handleFilterSubmit}
      >
        <label className="text-sm font-medium text-slate-700">
          搜索
          <input
            className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
            placeholder="按标题或描述搜索"
            value={queryInput}
            onChange={(event) => setQueryInput(event.target.value)}
          />
        </label>

        <label className="text-sm font-medium text-slate-700">
          状态
          <select
            className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
            value={statusInput}
            onChange={(event) => setStatusInput(event.target.value as TaskStatus | '')}
          >
            <option value="">全部状态</option>
            {statusOptions.map((status) => (
              <option key={status} value={status}>
                {statusLabel(status)}
              </option>
            ))}
          </select>
        </label>

        <div className="flex items-end gap-2">
          <button
            type="submit"
            className="rounded-md bg-slate-900 px-4 py-2 text-sm text-white"
          >
            查询
          </button>
          <button
            type="button"
            className="rounded-md border border-slate-300 px-4 py-2 text-sm text-slate-700"
            onClick={handleClearFilters}
          >
            清空
          </button>
        </div>
      </form>

      <div className="flex flex-col gap-2 text-sm text-slate-600 sm:flex-row sm:items-center sm:justify-between">
        <span>共 {list.total} 条任务</span>
        <span>
          第 {list.page} / {totalPages} 页
        </span>
      </div>

      <div className="overflow-x-auto rounded-xl border border-slate-200 bg-white">
        <table className="min-w-[720px] text-left text-sm sm:min-w-full">
          <thead className="bg-slate-50 text-slate-600">
            <tr>
              <th className="px-4 py-3">标题</th>
              <th className="px-4 py-3">状态</th>
              <th className="px-4 py-3">创建者</th>
              <th className="px-4 py-3">执行人</th>
              <th className="px-4 py-3">更新于</th>
            </tr>
          </thead>
          <tbody>
            {tasks.length === 0 ? (
              <tr>
                <td colSpan={5} className="px-4 py-8 text-center text-slate-500">
                  <EmptyState message="暂无任务" />
                </td>
              </tr>
            ) : (
              tasks.map((task) => (
                <tr key={task.id} className="border-t border-slate-100">
                  <td className="px-4 py-3 font-medium text-slate-900">
                    <Link to={`/tasks/${task.id}`} className="text-blue-600 hover:underline">
                      {task.title}
                    </Link>
                  </td>
                  <td className="px-4 py-3">{statusLabel(task.status)}</td>
                  <td className="px-4 py-3">{task.creator_id || '-'}</td>
                  <td className="px-4 py-3">{task.assignee_id || '-'}</td>
                  <td className="px-4 py-3">{new Date(task.updated_at).toLocaleString()}</td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      <div className="flex justify-end gap-2">
        <button
          type="button"
          className="rounded-md border border-slate-300 px-4 py-2 text-sm text-slate-700 disabled:cursor-not-allowed disabled:opacity-60"
          disabled={!canGoPrev}
          onClick={() => changePage(list.page - 1)}
        >
          上一页
        </button>
        <button
          type="button"
          className="rounded-md border border-slate-300 px-4 py-2 text-sm text-slate-700 disabled:cursor-not-allowed disabled:opacity-60"
          disabled={!canGoNext}
          onClick={() => changePage(list.page + 1)}
        >
          下一页
        </button>
      </div>
    </div>
  )
}
