// The shapes the Go API answers (internal/api). The states are the ones of
// docs/FLOW.md.

export const STATUSES = ['new', 'saved', 'applied', 'interview', 'discarded'] as const
export type Status = (typeof STATUSES)[number]

export interface Job {
  number: number
  title: string
  company: string
  url: string
  work_mode: string
  contract: string
  seniority: string
  location: string
  status: Status
  status_label: string
  note: string
  score: number
  new_today: boolean
  gone: boolean
  since: string
  first_seen: string
  last_seen: string
}

export interface StatusChange {
  at: string
  from: Status
  to: Status
}

export interface Posting {
  source: string
  location: string
  description: string
  tags: string[]
  work_mode: string
  contract: string
  seniority: string
  posted_at: string | null
}

export interface JobDetail extends Job {
  reasons: string[]
  history: StatusChange[]
  posting: Posting | null
}

export interface Stage {
  status: Status
  label: string
  count: number
  gone: number
}

export interface FunnelChange extends StatusChange {
  number: number
  title: string
  company: string
}

export interface Funnel {
  total: number
  stages: Stage[]
  history: FunnelChange[]
  last_run: string | null
}

export interface CollectState {
  available: boolean
  reason?: string
  running: boolean
  started_at: string | null
  finished_at: string | null
  error?: string
  log: string
}
