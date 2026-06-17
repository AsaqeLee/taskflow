import { EmptyState } from './EmptyState'
import type { AuditLog, TaskRecord } from '../types/api'

interface TaskTimelineTabsProps {
  tab: 'records' | 'audit'
  records: TaskRecord[]
  auditLogs: AuditLog[]
  onTabChange: (tab: 'records' | 'audit') => void
}

export function TaskTimelineTabs({
  tab,
  records,
  auditLogs,
  onTabChange,
}: TaskTimelineTabsProps) {
  return (
    <div className="rounded-xl border border-slate-200 bg-white">
      <div className="flex border-b border-slate-200">
        <button
          type="button"
          onClick={() => onTabChange('records')}
          className={`px-4 py-3 text-sm ${tab === 'records' ? 'border-b-2 border-blue-600 font-medium' : 'text-slate-600'}`}
        >
          记录
        </button>
        <button
          type="button"
          onClick={() => onTabChange('audit')}
          className={`px-4 py-3 text-sm ${tab === 'audit' ? 'border-b-2 border-blue-600 font-medium' : 'text-slate-600'}`}
        >
          审计
        </button>
      </div>
      <div className="divide-y divide-slate-100 p-4 text-sm">
        {tab === 'records' ? (
          records.length === 0 ? (
            <EmptyState message="暂无记录" />
          ) : (
            records.map((record) => (
              <div key={record.id} className="py-3">
                <p className="font-medium">
                  {record.type} · {record.author_id}
                </p>
                <p className="mt-1 text-slate-700">{record.content}</p>
                <p className="mt-1 text-xs text-slate-500">
                  {new Date(record.created_at).toLocaleString()}
                </p>
              </div>
            ))
          )
        ) : auditLogs.length === 0 ? (
          <EmptyState message="暂无审计日志" />
        ) : (
          auditLogs.map((log) => (
            <div key={log.id} className="py-3">
              <p className="font-medium">
                {log.action} · {log.actor_id}
              </p>
              <p className="mt-1 text-slate-600">
                {log.from_status} → {log.to_status}
              </p>
              <p className="mt-1 text-xs text-slate-500">
                {new Date(log.created_at).toLocaleString()}
              </p>
            </div>
          ))
        )}
      </div>
    </div>
  )
}