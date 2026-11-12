import { contractLabel, seniorityLabel, workModeLabel } from '../format'
import type { Job } from '../types'

/** What the posting says about where, how and at what level. */
export function Facts({ job }: { job: Pick<Job, 'work_mode' | 'contract' | 'seniority'> }) {
  return (
    <>
      <span className={`chip${job.work_mode ? '' : ' chip-unknown'}`}>
        {job.work_mode ? workModeLabel(job.work_mode) : 'modelo não informado'}
      </span>
      {job.contract && <span className="chip">{contractLabel(job.contract)}</span>}
      {job.seniority && <span className="chip">{seniorityLabel(job.seniority)}</span>}
    </>
  )
}

/** The state of the posting, with what the system marks by itself. */
export function State({ job }: { job: Pick<Job, 'status' | 'status_label' | 'new_today' | 'gone'> }) {
  return (
    <>
      {job.new_today && job.status === 'new' ? (
        <span className="chip chip-fresh">nova hoje</span>
      ) : (
        <span className={`chip chip-status chip-${job.status}`}>{job.status_label}</span>
      )}
      {job.gone && (
        <span className="chip chip-gone" title="A vaga não veio mais na coleta: a empresa pode ter fechado a posição">
          saiu da fonte
        </span>
      )}
    </>
  )
}
