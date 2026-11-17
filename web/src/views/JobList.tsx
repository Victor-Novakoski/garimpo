import { useState } from 'react'
import { listJobs } from '../api'
import { Facts, State } from '../components/Chips'
import { ScoreBadge } from '../components/ScoreBadge'
import { QuickActions } from '../components/StatusActions'
import { formatDay, safeUrl } from '../format'
import { FILTERS, filterHref, jobHref, type Filter } from '../router'
import type { Funnel, Job, Status } from '../types'
import { useLoad } from '../useLoad'

interface Props {
  filter: Filter
  version: number
  funnel: Funnel | null
  busy: ReadonlySet<number>
  onMark: (job: Job, to: Status) => void
}

function matches(job: Job, words: string[]): boolean {
  const text = `${job.title} ${job.company} #${job.number}`.toLocaleLowerCase('pt-BR')
  return words.every((word) => text.includes(word))
}

export function JobList({ filter, version, funnel, busy, onMark }: Props) {
  const { data, error, loading } = useLoad(() => listJobs(filter.statuses), `${filter.id}:${version}`)
  const [search, setSearch] = useState('')
  // While another filter loads, the rows of the previous one are not shown
  // under the new title.
  const [shown, setShown] = useState({ id: filter.id, jobs: data })
  if (!loading && shown.jobs !== data) setShown({ id: filter.id, jobs: data })
  const jobs = shown.id === filter.id || !loading ? data : null

  const words = search.toLocaleLowerCase('pt-BR').split(/\s+/).filter(Boolean)
  const visible = jobs?.filter((job) => matches(job, words)) ?? []
  const count = (status: Status | undefined) => funnel?.stages.find((stage) => stage.status === status)?.count

  return (
    <section aria-labelledby="list-title">
      <nav className="filters" aria-label="Filtrar por estado">
        {FILTERS.map((option) => (
          <a
            key={option.id}
            href={filterHref(option)}
            className="filter"
            aria-current={option === filter ? 'page' : undefined}
          >
            {option.label}
            {option.count && count(option.count) !== undefined && <span className="filter-count">{count(option.count)}</span>}
          </a>
        ))}
      </nav>

      <div className="list-head">
        <h1 id="list-title">
          {filter.label}
          {jobs && <span className="list-total">{visible.length === jobs.length ? jobs.length : `${visible.length} de ${jobs.length}`}</span>}
        </h1>
        <input
          type="search"
          className="search"
          placeholder="Buscar por vaga ou empresa"
          aria-label="Buscar por vaga ou empresa"
          value={search}
          onChange={(event) => setSearch(event.target.value)}
        />
      </div>

      {error && (
        <p className="notice notice-error" role="alert">
          {error}
        </p>
      )}
      {loading && !jobs && <p className="notice">Carregando as vagas…</p>}
      {jobs && jobs.length === 0 && !error && <p className="notice">{filter.empty}</p>}
      {jobs && jobs.length > 0 && visible.length === 0 && <p className="notice">Nenhuma vaga com "{search.trim()}" nesta lista.</p>}

      <ul className="jobs">
        {visible.map((job) => {
          const url = safeUrl(job.url)
          return (
            <li key={job.number} className={`job${job.status === 'saved' ? ' job-saved' : ''}`}>
              <ScoreBadge score={job.score} />
              <div className="job-main">
                <h2 className="job-title">
                  <a href={jobHref(job.number)}>{job.title || 'Vaga sem título'}</a>
                </h2>
                <p className="job-meta">
                  <span className="job-company">{job.company || 'empresa não informada'}</span>
                  <span className="job-number">#{job.number}</span>
                  {!filter.statuses || <span className="job-since">desde {formatDay(job.since)}</span>}
                </p>
                <p className="chips">
                  <State job={job} />
                  <Facts job={job} />
                </p>
                {job.note && <p className="job-note">{job.note}</p>}
              </div>
              <div className="job-side">
                <QuickActions number={job.number} status={job.status} busy={busy.has(job.number)} onMark={(to) => onMark(job, to)} />
                {url && (
                  <a className="external" href={url} target="_blank" rel="noopener noreferrer">
                    Abrir a vaga<span aria-hidden="true"> ↗</span>
                    <span className="visually-hidden"> {job.title} (abre em outra aba)</span>
                  </a>
                )}
              </div>
            </li>
          )
        })}
      </ul>
    </section>
  )
}
