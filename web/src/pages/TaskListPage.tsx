import { Link } from 'react-router-dom'
import { EmptyState } from '../components/EmptyState'
import { ErrorAlert } from '../components/ErrorAlert'
import { LoadingState } from '../components/LoadingState'
import { statusLabel } from '../domain/taskWorkflow'
import { useAsyncData } from '../hooks/useAsyncData'
import { fetchTasks } from '../lib/apiClient'

export function TaskListPage() {
  const tasksState = useAsyncData('tasks', fetchTasks)

  if (tasksState.status === 'idle' || tasksState.status === 'loading') {
    return <LoadingState />
  }

  const tasks = tasksState.data ?? []
  const error = tasksState.error

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
                <tr key={task.id} className="border-t border-slate-100 hover:bg-slate-50">
                  <td className="px-4 py-3">
                    <Link to={`/tasks/${task.id}`} className="font-medium text-blue-700">
                      {task.title}
                    </Link>
                  </td>
                  <td className="px-4 py-3">{statusLabel(task.status)}</td>
                  <td className="px-4 py-3">{task.creator_id}</td>
                  <td className="px-4 py-3">{task.assignee_id || '—'}</td>
                  <td className="px-4 py-3">
                    {new Date(task.updated_at).toLocaleString()}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}
