import { useCallback, useEffect, useRef, useState } from 'react'
import { getFunnel, updateJob } from './api'
import { CollectButton, CollectPanel, useCollect } from './components/Collect'
import { Toasts, type Toast } from './components/Toasts'
import { STATUS_LABEL, formatMoment } from './format'
import { DAILY, filterHref, useRoute } from './router'
import type { Job, Status } from './types'
import { useLoad } from './useLoad'
import { FunnelView } from './views/FunnelView'
import { JobDetail } from './views/JobDetail'
import { JobList } from './views/JobList'

const TOAST_MS = 8000

interface Props {
  /** How often a running collection is checked; tests make it short. */
  pollMs?: number
}

export function App({ pollMs }: Props) {
  const route = useRoute()
  // Every change (a mark, a note, a collection) bumps the version, and the
  // screens load again from the server: it is the one that knows the rules.
  const [version, setVersion] = useState(0)
  const refresh = useCallback(() => setVersion((v) => v + 1), [])
  const [busy, setBusy] = useState<ReadonlySet<number>>(new Set())
  const [toasts, setToasts] = useState<Toast[]>([])
  const nextToast = useRef(1)
  const timers = useRef(new Map<number, number>())

  const dismiss = useCallback((id: number) => {
    window.clearTimeout(timers.current.get(id))
    timers.current.delete(id)
    setToasts((all) => all.filter((toast) => toast.id !== id))
  }, [])

  const notify = useCallback(
    (toast: Omit<Toast, 'id'>) => {
      const id = nextToast.current++
      setToasts((all) => [...all.slice(-2), { ...toast, id }])
      timers.current.set(
        id,
        window.setTimeout(() => dismiss(id), TOAST_MS),
      )
    },
    [dismiss],
  )
  useEffect(() => {
    const pending = timers.current
    return () => pending.forEach((timer) => window.clearTimeout(timer))
  }, [])

  const collect = useCollect(refresh, pollMs)
  const funnel = useLoad(getFunnel, `funnel:${version}`)

  // Where "voltar" goes from a posting: the last list the person saw.
  const here = route.page === 'list' ? filterHref(route.filter) : route.page === 'funnel' ? '#/funil' : null
  const [backHref, setBackHref] = useState(here ?? filterHref(DAILY))
  if (here !== null && here !== backHref) setBackHref(here)

  /** Marks and reloads. It answers whether the server took the change. */
  async function setStatus(number: number, to: Status): Promise<boolean> {
    setBusy((all) => new Set(all).add(number))
    try {
      await updateJob(number, { status: to })
      return true
    } catch (err) {
      notify({ kind: 'error', text: err instanceof Error ? err.message : 'Não consegui marcar a vaga.' })
      return false
    } finally {
      setBusy((all) => {
        const rest = new Set(all)
        rest.delete(number)
        return rest
      })
      refresh()
    }
  }

  async function mark(job: Pick<Job, 'number' | 'status'>, to: Status) {
    if (!(await setStatus(job.number, to))) return
    notify({
      kind: 'done',
      text: `Vaga #${job.number} marcada como "${STATUS_LABEL[to]}".`,
      // Undoing puts back the state it had, and is not undone again.
      undo: () => {
        void setStatus(job.number, job.status).then(
          (ok) => ok && notify({ kind: 'done', text: `Desfeito: a vaga #${job.number} voltou para "${STATUS_LABEL[job.status]}".` }),
        )
      },
    })
  }

  return (
    <>
      <a className="skip" href="#conteudo">
        Pular para o conteúdo
      </a>
      <header className="top">
        <div className="top-row">
          <a className="brand" href="#/">
            <span className="brand-mark" aria-hidden="true" />
            garimpo
          </a>
          <nav className="tabs" aria-label="Telas">
            <a href={route.page === 'list' ? filterHref(route.filter) : '#/'} aria-current={route.page === 'list' ? 'page' : undefined}>
              Vagas
            </a>
            <a href="#/funil" aria-current={route.page === 'funnel' ? 'page' : undefined}>
              Funil
            </a>
          </nav>
          <CollectButton collect={collect} />
        </div>
        <p className="top-status">
          {funnel.data?.last_run
            ? `Última coleta em ${formatMoment(funnel.data.last_run)}`
            : funnel.data
              ? 'Nenhuma coleta ainda'
              : ''}
          <span className="top-local">só nesta máquina</span>
        </p>
      </header>

      <main id="conteudo" className="page">
        <CollectPanel collect={collect} />
        {route.page === 'list' && (
          <JobList
            filter={route.filter}
            version={version}
            funnel={funnel.data}
            busy={busy}
            onMark={(job, to) => void mark(job, to)}
          />
        )}
        {route.page === 'job' && (
          <JobDetail
            key={route.number}
            number={route.number}
            backHref={backHref}
            version={version}
            busy={busy}
            onMark={(job, to) => void mark(job, to)}
            onSaved={(text) => {
              notify({ kind: 'done', text })
              refresh()
            }}
            onError={(text) => notify({ kind: 'error', text })}
          />
        )}
        {route.page === 'funnel' && <FunnelView funnel={funnel.data} error={funnel.error} />}
        {route.page === 'missing' && (
          <section>
            <h1>Esta tela não existe</h1>
            <p className="notice">
              O endereço não leva a lugar nenhum. <a href="#/">Voltar para a lista do dia</a>.
            </p>
          </section>
        )}
      </main>
      <Toasts toasts={toasts} dismiss={dismiss} />
    </>
  )
}
