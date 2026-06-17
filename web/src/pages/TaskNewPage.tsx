import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { ErrorAlert } from '../components/ErrorAlert'
import { createTask } from '../lib/apiClient'

export function TaskNewPage() {
  const navigate = useNavigate()
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [error, setError] = useState<unknown>(null)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    setLoading(true)
    setError(null)
    try {
      const task = await createTask(title.trim(), description.trim())
      navigate(`/tasks/${task.id}`)
    } catch (err) {
      setError(err)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="mx-auto max-w-xl space-y-4">
      <h1 className="text-2xl font-semibold">新建任务</h1>
      <form
        onSubmit={handleSubmit}
        className="space-y-4 rounded-xl border border-slate-200 bg-white p-6"
      >
        <div>
          <label htmlFor="task-title" className="text-sm font-medium">
            标题
          </label>
          <input
            id="task-title"
            className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            minLength={3}
            required
          />
        </div>
        <div>
          <label htmlFor="task-description" className="text-sm font-medium">
            描述
          </label>
          <textarea
            id="task-description"
            className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
            rows={5}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </div>
        <ErrorAlert error={error} />
        <button
          type="submit"
          disabled={loading}
          className="rounded-md bg-blue-600 px-4 py-2 text-white disabled:opacity-50"
        >
          {loading ? '创建中…' : '创建'}
        </button>
      </form>
    </div>
  )
}
