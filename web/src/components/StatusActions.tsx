import { STATUS_LABEL } from '../format'
import { STATUSES, type Status } from '../types'

// The usual next steps from each state (the diagram of docs/FLOW.md). Any
// state can still become any other on the posting's own screen.
const NEXT: Record<Status, { to: Status; label: string }[]> = {
  new: [
    { to: 'saved', label: 'Vou aplicar' },
    { to: 'applied', label: 'Apliquei' },
    { to: 'discarded', label: 'Descartar' },
  ],
  saved: [
    { to: 'applied', label: 'Apliquei' },
    { to: 'discarded', label: 'Descartar' },
  ],
  applied: [
    { to: 'interview', label: 'Entrevista' },
    { to: 'discarded', label: 'Descartar' },
  ],
  interview: [{ to: 'discarded', label: 'Descartar' }],
  discarded: [{ to: 'new', label: 'Voltar para nova' }],
}

interface Props {
  number: number
  status: Status
  busy: boolean
  onMark: (to: Status) => void
}

/** One click for the usual next step, in a list row. */
export function QuickActions({ number, status, busy, onMark }: Props) {
  return (
    <div className="actions" role="group" aria-label={`Marcar a vaga ${number}`}>
      {NEXT[status].map(({ to, label }) => (
        <button
          key={to}
          type="button"
          className={`button button-mark${to === 'discarded' ? ' button-discard' : ''}`}
          disabled={busy}
          onClick={() => onMark(to)}
        >
          {label}
        </button>
      ))}
    </div>
  )
}

/** Every state, on the posting's screen: marking wrong happens. */
export function StatusPicker({ number, status, busy, onMark }: Props) {
  return (
    <div className="picker" role="group" aria-label={`Estado da vaga ${number}`}>
      {STATUSES.map((to) => (
        <button
          key={to}
          type="button"
          className="button picker-option"
          aria-pressed={to === status}
          disabled={busy}
          onClick={() => to !== status && onMark(to)}
        >
          {STATUS_LABEL[to]}
        </button>
      ))}
    </div>
  )
}
