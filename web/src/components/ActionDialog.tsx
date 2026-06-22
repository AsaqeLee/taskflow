import { useId, useState } from 'react'
import type { TaskAction, TaskRecordInput, TaskRecordMetadataField } from '../types/api'
import { actionLabel } from '../domain/taskWorkflow'

interface ActionDialogProps {
  action: TaskAction | null
  needsContent: boolean
  metadataFields?: TaskRecordMetadataField[]
  onClose: () => void
  onConfirm: (payload: TaskRecordInput) => Promise<void>
}

export function ActionDialog({
  action,
  needsContent,
  metadataFields = [],
  onClose,
  onConfirm,
}: ActionDialogProps) {
  const [content, setContent] = useState('')
  const [metadata, setMetadata] = useState<Record<string, string>>({})
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const titleId = useId()
  const descriptionId = useId()

  if (!action) return null

  const destructive = action === 'delete'

  async function handleSubmit(event: React.FormEvent) {
    event.preventDefault()
    if (needsContent && !content.trim()) {
      setError('请填写说明内容')
      return
    }
    setSubmitting(true)
    setError(null)
    try {
      await onConfirm({
        content: content.trim(),
        metadata: sanitizeMetadata(metadata),
      })
      setContent('')
      setMetadata({})
      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : '操作失败')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <form
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={descriptionId}
        onSubmit={handleSubmit}
        className="w-full max-w-md rounded-xl bg-white p-5 shadow-xl"
      >
        <h3 id={titleId} className="text-lg font-semibold">
          {actionLabel(action)}
        </h3>
        {needsContent ? (
          <div className="mt-4 space-y-3">
            <textarea
              className="w-full rounded-md border border-slate-300 px-3 py-2 text-sm"
              rows={4}
              placeholder="填写说明（必填）"
              value={content}
              onChange={(e) => setContent(e.target.value)}
            />
            {metadataFields.map((field) => (
              <div key={field.key}>
                <label className="mb-1 block text-sm font-medium text-slate-700">
                  {field.label}
                </label>
                <textarea
                  className="w-full rounded-md border border-slate-300 px-3 py-2 text-sm"
                  rows={2}
                  placeholder={field.placeholder}
                  value={metadata[field.key] ?? ''}
                  onChange={(e) =>
                    setMetadata((current) => ({
                      ...current,
                      [field.key]: e.target.value,
                    }))
                  }
                />
              </div>
            ))}
          </div>
        ) : (
          <p id={descriptionId} className="mt-2 text-sm text-slate-600">
            确认执行此操作？
          </p>
        )}
        {error ? <p className="mt-2 text-sm text-red-600">{error}</p> : null}
        <div className="mt-4 flex justify-end gap-2">
          <button
            type="button"
            onClick={onClose}
            className="rounded-md border border-slate-300 px-3 py-1.5 text-sm"
          >
            取消
          </button>
          <button
            type="submit"
            disabled={submitting}
            className={`rounded-md px-3 py-1.5 text-sm text-white disabled:opacity-50 ${
              destructive ? 'bg-red-600' : 'bg-blue-600'
            }`}
          >
            {submitting ? '处理中…' : '确认'}
          </button>
        </div>
      </form>
    </div>
  )
}

function sanitizeMetadata(metadata: Record<string, string>): Record<string, string> | undefined {
  const normalized = Object.entries(metadata).reduce<Record<string, string>>((result, [key, value]) => {
    const trimmedValue = value.trim()
    if (trimmedValue) {
      result[key] = trimmedValue
    }
    return result
  }, {})

  return Object.keys(normalized).length > 0 ? normalized : undefined
}
