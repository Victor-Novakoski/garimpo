import type { CollectState, Funnel, Job, JobDetail, Status } from './types'

// The panel and the API come from the same `garimpo serve`, so every call
// is same-origin and relative. There is no token: the server only answers
// its own page, on this machine.

export class ApiError extends Error {
  readonly status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

const OFFLINE = 'Não consegui falar com o Garimpo. Ele ainda está rodando no terminal?'

async function call<T>(path: string, init?: RequestInit, accept: number[] = []): Promise<T> {
  let res: Response
  try {
    res = await fetch(`/api${path}`, init)
  } catch {
    throw new ApiError(0, OFFLINE)
  }
  let body: unknown = null
  try {
    body = await res.json()
  } catch {
    // An answer without JSON is handled below, by the status.
  }
  if (!res.ok && !accept.includes(res.status)) {
    const message =
      body !== null && typeof body === 'object' && 'error' in body && typeof body.error === 'string'
        ? body.error
        : `O Garimpo respondeu com erro (${res.status}).`
    throw new ApiError(res.status, message)
  }
  return body as T
}

/** The daily list when no status is given; otherwise every posting in those states. */
export async function listJobs(statuses?: readonly Status[] | 'all'): Promise<Job[]> {
  const query = statuses === undefined ? '' : `?status=${statuses === 'all' ? 'all' : statuses.join(',')}`
  return (await call<{ jobs: Job[] }>(`/jobs${query}`)).jobs
}

export function getJob(number: number): Promise<JobDetail> {
  return call<JobDetail>(`/jobs/${number}`)
}

export function updateJob(number: number, change: { status?: Status; note?: string }): Promise<JobDetail> {
  return call<JobDetail>(`/jobs/${number}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(change),
  })
}

export function getFunnel(): Promise<Funnel> {
  return call<Funnel>('/funnel')
}

export function getCollect(): Promise<CollectState> {
  return call<CollectState>('/collect')
}

/** Starts a collection. A 409 (already running, or unavailable) still answers the state. */
export function startCollect(): Promise<CollectState> {
  return call<CollectState>('/collect', { method: 'POST' }, [409])
}
