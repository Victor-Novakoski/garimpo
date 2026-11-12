import { STATUS_LABEL, formatMoment } from '../format'
import { jobHref, statusHref } from '../router'
import type { Funnel } from '../types'

interface Props {
  funnel: Funnel | null
  error: string | null
}

export function FunnelView({ funnel, error }: Props) {
  if (!funnel) {
    return (
      <section aria-labelledby="funnel-title">
        <h1 id="funnel-title">Funil</h1>
        {error ? (
          <p className="notice notice-error" role="alert">
            {error}
          </p>
        ) : (
          <p className="notice">Carregando o funil…</p>
        )}
      </section>
    )
  }
  const most = Math.max(1, ...funnel.stages.map((stage) => stage.count))
  return (
    <section aria-labelledby="funnel-title">
      <h1 id="funnel-title">Funil</h1>
      <p className="hint">
        {funnel.total === 1 ? '1 vaga no registro' : `${funnel.total} vagas no registro`}
        {funnel.last_run && `, última coleta em ${formatMoment(funnel.last_run)}`}. Clique num estado para ver as vagas.
      </p>

      <ol className="funnel">
        {funnel.stages.map((stage) => (
          <li key={stage.status}>
            <a className="stage" href={statusHref(stage.status)}>
              <span className="stage-label">{stage.label}</span>
              <span className="stage-track" aria-hidden="true">
                <span className="stage-bar" style={{ width: `${(stage.count / most) * 100}%` }} />
              </span>
              <span className="stage-count">
                {stage.count}
                <span className="visually-hidden"> {stage.count === 1 ? 'vaga' : 'vagas'}</span>
              </span>
              {stage.gone > 0 && (
                <span className="stage-gone">
                  {stage.gone === 1 ? '1 saiu da fonte' : `${stage.gone} saíram da fonte`}
                </span>
              )}
            </a>
          </li>
        ))}
      </ol>

      <h2>Últimas mudanças</h2>
      {funnel.history.length === 0 ? (
        <p className="notice">Nada marcado ainda. Quando você marcar uma vaga, a mudança aparece aqui com a data.</p>
      ) : (
        <ol className="history history-all">
          {funnel.history.map((change, index) => (
            <li key={index}>
              <time dateTime={change.at}>{formatMoment(change.at)}</time>
              <span>
                <a href={jobHref(change.number)}>
                  #{change.number} {change.title}
                </a>
                {change.company && <span className="job-company"> · {change.company}</span>}
                <br />
                {STATUS_LABEL[change.from]} → <strong>{STATUS_LABEL[change.to]}</strong>
              </span>
            </li>
          ))}
        </ol>
      )}
    </section>
  )
}
