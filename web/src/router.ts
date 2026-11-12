import { useEffect, useState } from 'react'
import type { Status } from './types'

// The panel is served as one page, so the screens live in the hash:
// #/ (the daily list), #/vagas/apliquei, #/vaga/12, #/funil.

export interface Filter {
  id: string
  label: string
  /** undefined is the daily list of docs/FLOW.md. */
  statuses: readonly Status[] | 'all' | undefined
  /** The state whose count goes beside the label. */
  count?: Status
  empty: string
}

export const FILTERS: readonly Filter[] = [
  { id: 'hoje', label: 'Lista do dia', statuses: undefined, empty: 'Nada para decidir agora. Busque vagas para ver se chegou algo novo.' },
  { id: 'vou-aplicar', label: 'Vou aplicar', statuses: ['saved'], count: 'saved', empty: 'Nenhuma vaga marcada como "vou aplicar".' },
  { id: 'apliquei', label: 'Apliquei', statuses: ['applied'], count: 'applied', empty: 'Você ainda não marcou nenhuma candidatura.' },
  { id: 'entrevista', label: 'Entrevista', statuses: ['interview'], count: 'interview', empty: 'Nenhuma entrevista por enquanto.' },
  { id: 'andamento', label: 'Em andamento', statuses: ['interview', 'applied', 'saved'], empty: 'Nada em andamento.' },
  { id: 'descartadas', label: 'Descartadas', statuses: ['discarded'], count: 'discarded', empty: 'Nenhuma vaga descartada.' },
  { id: 'todas', label: 'Todas', statuses: 'all', empty: 'O Garimpo ainda não achou nenhuma vaga. Busque vagas para começar.' },
]

export const DAILY = FILTERS[0] as Filter

export type Route =
  | { page: 'list'; filter: Filter }
  | { page: 'job'; number: number }
  | { page: 'funnel' }
  | { page: 'missing' }

export function parseRoute(hash: string): Route {
  const parts = hash.replace(/^#\/?/, '').split('/').filter(Boolean)
  const [first, second] = parts
  if (first === undefined) return { page: 'list', filter: DAILY }
  if (first === 'funil' && parts.length === 1) return { page: 'funnel' }
  if (first === 'vagas' && parts.length === 2) {
    const filter = FILTERS.find((f) => f.id === second)
    if (filter) return { page: 'list', filter }
  }
  if (first === 'vaga' && parts.length === 2 && /^[1-9]\d{0,8}$/.test(second ?? '')) {
    return { page: 'job', number: Number(second) }
  }
  return { page: 'missing' }
}

export const filterHref = (filter: Filter): string => (filter === DAILY ? '#/' : `#/vagas/${filter.id}`)
export const jobHref = (number: number): string => `#/vaga/${number}`

/** The filter that shows one state, for the links of the funnel. */
export function statusHref(status: Status): string {
  if (status === 'new') return filterHref(DAILY)
  const filter = FILTERS.find((f) => f.count === status)
  return filter ? filterHref(filter) : '#/'
}

export function useRoute(): Route {
  const [hash, setHash] = useState(() => window.location.hash)
  useEffect(() => {
    const onChange = () => setHash(window.location.hash)
    window.addEventListener('hashchange', onChange)
    return () => window.removeEventListener('hashchange', onChange)
  }, [])
  return parseRoute(hash)
}
