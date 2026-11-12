import { vi } from 'vitest'
import type { CollectState, Funnel, Job, JobDetail, Status } from '../types'
import { STATUSES } from '../types'

// A small stand-in for the Go API, with the rules the panel depends on:
// the daily list is "vou aplicar" first and then by score, without what was
// applied to, discarded or left its source.

const LABEL: Record<Status, string> = {
  new: 'nova',
  saved: 'vou aplicar',
  applied: 'apliquei',
  interview: 'entrevista',
  discarded: 'descartada',
}

export function job(number: number, over: Partial<JobDetail> = {}): JobDetail {
  const status = over.status ?? 'new'
  return {
    number,
    title: `Vaga ${number}`,
    company: 'Acme',
    url: `https://example.com/vagas/${number}`,
    work_mode: 'remote',
    contract: 'pj',
    seniority: 'mid',
    location: 'Brasil',
    note: '',
    score: 60,
    new_today: false,
    gone: false,
    since: '2026-11-11T12:00:00Z',
    first_seen: '2026-11-10T12:00:00Z',
    last_seen: '2026-11-12T12:00:00Z',
    reasons: ['+15 Go no título'],
    history: [],
    posting: null,
    ...over,
    status,
    status_label: LABEL[status],
  }
}

export interface FakeApi {
  jobs: JobDetail[]
  collect: CollectState
  /** Every request made, as "METHOD /path body". */
  calls: string[]
  /** When set, the next request fails with this. */
  failNext: { status: number; error: string } | null
  /** Called when a collection starts, to say how it goes on. */
  onCollect: (api: FakeApi) => void
}

function answer(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

function listed(api: FakeApi, statusParam: string | null): Job[] {
  if (statusParam === null) {
    return api.jobs
      .filter((j) => (j.status === 'new' || j.status === 'saved') && !j.gone)
      .sort((a, b) => Number(b.status === 'saved') - Number(a.status === 'saved') || b.score - a.score)
  }
  const want = statusParam === 'all' ? [...STATUSES] : (statusParam.split(',') as Status[])
  return api.jobs.filter((j) => want.includes(j.status)).sort((a, b) => want.indexOf(a.status) - want.indexOf(b.status))
}

function funnel(api: FakeApi): Funnel {
  return {
    total: api.jobs.length,
    stages: STATUSES.map((status) => ({
      status,
      label: LABEL[status],
      count: api.jobs.filter((j) => j.status === status).length,
      gone: api.jobs.filter((j) => j.status === status && j.gone).length,
    })),
    history: api.jobs
      .flatMap((j) => j.history.map((h) => ({ ...h, number: j.number, title: j.title, company: j.company })))
      .sort((a, b) => b.at.localeCompare(a.at)),
    last_run: api.jobs.length > 0 ? '2026-11-12T12:00:00Z' : null,
  }
}

/** Replaces fetch with the fake API and returns it, to arrange and inspect. */
export function fakeApi(jobs: JobDetail[] = []): FakeApi {
  const api: FakeApi = {
    jobs,
    collect: { available: true, running: false, started_at: null, finished_at: null, log: '' },
    calls: [],
    failNext: null,
    onCollect: () => {},
  }
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = new URL(String(input), 'http://127.0.0.1:8080')
      const method = init?.method ?? 'GET'
      const body = typeof init?.body === 'string' ? init.body : ''
      api.calls.push(`${method} ${url.pathname}${url.search}${body ? ` ${body}` : ''}`)
      if (api.failNext) {
        const { status, error } = api.failNext
        api.failNext = null
        return answer(status, { error })
      }
      const path = url.pathname.replace(/^\/api/, '')
      if (path === '/jobs' && method === 'GET') return answer(200, { jobs: listed(api, url.searchParams.get('status')) })
      if (path === '/funnel') return answer(200, funnel(api))
      if (path === '/collect' && method === 'GET') return answer(200, api.collect)
      if (path === '/collect' && method === 'POST') {
        if (api.collect.running || !api.collect.available) return answer(409, api.collect)
        api.collect = { ...api.collect, running: true, started_at: '2026-11-12T13:00:00Z', finished_at: null, log: '', error: undefined }
        api.onCollect(api)
        return answer(202, api.collect)
      }
      const match = /^\/jobs\/(\d+)$/.exec(path)
      const found = match ? api.jobs.find((j) => j.number === Number(match[1])) : undefined
      if (match && !found) return answer(404, { error: `não existe vaga #${match[1]}` })
      if (found && method === 'GET') return answer(200, found)
      if (found && method === 'PATCH') {
        const change = JSON.parse(body) as { status?: Status; note?: string }
        if (change.status && change.status !== found.status) {
          found.history = [...found.history, { at: '2026-11-12T14:00:00Z', from: found.status, to: change.status }]
          found.status = change.status
          found.status_label = LABEL[change.status]
        }
        if (change.note !== undefined) found.note = change.note.trim()
        return answer(200, found)
      }
      return answer(404, { error: 'rota desconhecida' })
    }),
  )
  return api
}
