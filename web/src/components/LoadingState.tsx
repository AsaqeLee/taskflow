interface LoadingStateProps {
  label?: string
}

export function LoadingState({ label = '加载中…' }: LoadingStateProps) {
  return <p className="text-sm text-slate-600">{label}</p>
}