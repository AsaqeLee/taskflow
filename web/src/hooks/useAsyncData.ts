import { useCallback, useEffect, useState } from 'react'

export type AsyncStatus = 'idle' | 'loading' | 'success' | 'error'

export interface AsyncState<T> {
  data: T | null
  error: unknown
  status: AsyncStatus
  reload: () => void
}

interface Snapshot<T> {
  key: string
  generation: number
  data: T | null
  error: unknown
}

export function useAsyncData<T>(key: string, loader: () => Promise<T>): AsyncState<T> {
  const [generation, setGeneration] = useState(0)
  const [snapshot, setSnapshot] = useState<Snapshot<T>>({
    key,
    generation: -1,
    data: null,
    error: null,
  })

  const reload = useCallback(() => {
    setGeneration((value) => value + 1)
  }, [])

  useEffect(() => {
    const currentGen = generation
    let cancelled = false

    loader()
      .then((data) => {
        if (cancelled) return
        setSnapshot({ key, generation: currentGen, data, error: null })
      })
      .catch((error) => {
        if (cancelled) return
        setSnapshot({ key, generation: currentGen, data: null, error })
      })

    return () => {
      cancelled = true
    }
  }, [key, generation, loader])

  const pending = snapshot.generation !== generation || snapshot.key !== key
  const status: AsyncStatus = pending ? 'loading' : snapshot.error ? 'error' : 'success'

  return {
    data: pending ? null : snapshot.data,
    error: pending ? null : snapshot.error,
    status,
    reload,
  }
}