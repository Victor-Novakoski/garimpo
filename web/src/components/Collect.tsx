import { useEffect, useRef, useState } from 'react'
import { getCollect, startCollect } from '../api'
import { formatMoment } from '../format'
import type { CollectState } from '../types'

const POLL_MS = 1500

/**
 * The button that runs a collection and the box that follows it. The
 * collection is the one `garimpo serve` was started with; the panel only
 * starts it and watches.
 */
export function useCollect(onFinished: () => void, pollMs = POLL_MS) {
  const [state, setState] = useState<CollectState | null>(null)
  const [failure, setFailure] = useState<string | null>(null)
  const [open, setOpen] = useState(false)
  const finished = useRef(onFinished)
  useEffect(() => {
    finished.current = onFinished
  }, [onFinished])

  // A collection may already be running: started in another tab, or before
  // this page was reloaded.
  useEffect(() => {
    let current = true
    getCollect().then(
      (now) => {
        if (!current) return
        setState(now)
        if (now.running) setOpen(true)
      },
      () => {}, // the lists already say when the server is unreachable
    )
    return () => {
      current = false
    }
  }, [])

  const running = state?.running ?? false
  useEffect(() => {
    if (!running) return
    let current = true
    const timer = window.setInterval(() => {
      getCollect().then(
        (now) => {
          if (!current) return
          setState(now)
          setFailure(null)
          if (!now.running) finished.current()
        },
        (err: unknown) => current && setFailure(err instanceof Error ? err.message : 'Algo deu errado.'),
      )
    }, pollMs)
    return () => {
      current = false
      window.clearInterval(timer)
    }
  }, [running, pollMs])

  async function start() {
    setOpen(true)
    setFailure(null)
    try {
      setState(await startCollect())
    } catch (err) {
      setFailure(err instanceof Error ? err.message : 'Algo deu errado.')
    }
  }

  return { state, failure, open, running, start, close: () => setOpen(false) }
}

export type Collect = ReturnType<typeof useCollect>

export function CollectButton({ collect }: { collect: Collect }) {
  const unavailable = collect.state !== null && !collect.state.available
  return (
    <button
      type="button"
      className="button button-primary"
      onClick={collect.start}
      disabled={collect.running || unavailable}
      title={unavailable ? collect.state?.reason : 'Coleta as fontes agora e atualiza a lista'}
    >
      {collect.running ? 'Buscando vagas…' : 'Buscar vagas'}
    </button>
  )
}

export function CollectPanel({ collect }: { collect: Collect }) {
  const { state, failure } = collect
  const log = useRef<HTMLPreElement>(null)
  const text = state?.log ?? ''
  useEffect(() => {
    // Follow the end of the log as it grows.
    if (log.current && text !== '') log.current.scrollTop = log.current.scrollHeight
  }, [text])

  if (!collect.open || state === null) {
    return failure && collect.open ? (
      <section className="collect collect-failed" aria-label="Coleta">
        <p role="alert">{failure}</p>
      </section>
    ) : null
  }
  const error = failure ?? state.error ?? (state.available ? null : (state.reason ?? null))
  let headline = 'Coleta'
  if (state.running) headline = 'Buscando vagas nas fontes. Pode levar alguns minutos.'
  else if (error) headline = 'A coleta não terminou'
  else if (state.finished_at) headline = `Coleta concluída em ${formatMoment(state.finished_at)}`

  return (
    <section className={`collect${error && !state.running ? ' collect-failed' : ''}`} aria-label="Coleta">
      <div className="collect-head">
        <p className="collect-headline" role="status">
          {state.running && <span className="spinner" aria-hidden="true" />}
          {headline}
        </p>
        {!state.running && (
          <button type="button" className="button button-quiet" onClick={collect.close}>
            Fechar
          </button>
        )}
      </div>
      {error && <p role="alert">{error}</p>}
      {state.log !== '' && (
        // The log scrolls, so the keyboard must be able to reach it.
        // oxlint-disable-next-line jsx-a11y/no-noninteractive-tabindex
        <pre ref={log} className="collect-log" tabIndex={0} aria-label="Andamento da coleta">
          {state.log}
        </pre>
      )}
    </section>
  )
}
