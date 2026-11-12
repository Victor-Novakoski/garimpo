import { useEffect, useState } from 'react'

export interface Loaded<T> {
  data: T | null
  error: string | null
  /** True until the first answer; a refresh keeps showing what was there. */
  loading: boolean
}

/**
 * Loads on mount and again whenever key changes. While a refresh runs the
 * previous data stays on screen, so marking a posting does not blank the
 * list. An answer that arrives after the key changed is dropped.
 */
export function useLoad<T>(load: () => Promise<T>, key: string): Loaded<T> {
  const [state, setState] = useState<Loaded<T> & { key: string }>({ data: null, error: null, loading: true, key })
  useEffect(() => {
    let current = true
    load().then(
      (data) => current && setState({ data, error: null, loading: false, key }),
      (err: unknown) =>
        current &&
        setState((before) => ({
          data: before.data,
          error: err instanceof Error ? err.message : 'Algo deu errado.',
          loading: false,
          key,
        })),
    )
    return () => {
      current = false
    }
    // load is rebuilt on every render; key says when it means something new.
    // oxlint-disable-next-line react-hooks/exhaustive-deps
  }, [key])
  return state
}
