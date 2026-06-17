import type { Task } from '../types/api'

interface TaskBasicInfoPanelProps {
  task: Task
  canEdit: boolean
  editing: boolean
  title: string
  description: string
  onToggleEdit: () => void
  onTitleChange: (value: string) => void
  onDescriptionChange: (value: string) => void
  onSave: () => void
}

export function TaskBasicInfoPanel({
  task,
  canEdit,
  editing,
  title,
  description,
  onToggleEdit,
  onTitleChange,
  onDescriptionChange,
  onSave,
}: TaskBasicInfoPanelProps) {
  if (!canEdit) return null

  return (
    <div className="rounded-xl border border-slate-200 bg-white p-4">
      <div className="flex items-center justify-between">
        <h2 className="font-medium">基本信息</h2>
        <button
          type="button"
          onClick={onToggleEdit}
          className="text-sm text-blue-700"
        >
          {editing ? '取消' : '编辑'}
        </button>
      </div>
      {editing ? (
        <div className="mt-3 space-y-3">
          <input
            className="w-full rounded-md border border-slate-300 px-3 py-2"
            value={title}
            onChange={(e) => onTitleChange(e.target.value)}
          />
          <textarea
            className="w-full rounded-md border border-slate-300 px-3 py-2"
            rows={4}
            value={description}
            onChange={(e) => onDescriptionChange(e.target.value)}
          />
          <button
            type="button"
            onClick={onSave}
            className="rounded-md bg-blue-600 px-3 py-1.5 text-sm text-white"
          >
            保存
          </button>
        </div>
      ) : (
        <p className="mt-2 text-xs text-slate-500">创建者：{task.creator_id}</p>
      )}
    </div>
  )
}