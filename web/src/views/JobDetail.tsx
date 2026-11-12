import { useState } from 'react'
import { getJob, updateJob } from '../api'
import { Facts, State } from '../components/Chips'
import { ScoreBadge } from '../components/ScoreBadge'
import { StatusPicker } from '../components/StatusActions'
import { STATUS_LABEL, formatDay, formatMoment, safeUrl, splitReasons } from '../format'
import type { Job, Status } from '../types'
import { useLoad } from '../useLoad'

const MAX_NOTE = 4000 // the limit of the API

interface Props {
  number: number
  /** The list the person came from. */
  backHref: string
  version: number
  busy: ReadonlySet<number>
  onMark: (job: Job, to: Status) => void
  onSaved: (text: string) => void
  onError: (text: string) => void
}

export function JobDetail({ number, backHref, version, busy, onMark, onSaved, onError }: Props) {
  const { data: job, error, loading } = useLoad(() => getJob(number), `${number}:${version}`)
  // null: the person did not touch the note, so it shows what is saved.
  const [draft, setDraft] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  if (error && !job) {
    return (
      <section>
        <BackLink href={backHref} />
        <p className="notice notice-error" role="alert">
          {error}
        </p>
      </section>
    )
  }
  if (!job || job.number !== number) {
    return (
      <section>
        <BackLink href={backHref} />
        {loading && <p className="notice">Carregando a vaga…</p>}
      </section>
    )
  }

  const url = safeUrl(job.url)
  const reasons = splitReasons(job.reasons)
  const note = draft ?? job.note
  const changed = draft !== null && draft.trim() !== job.note

  async function saveNote() {
    if (!changed || saving) return
    setSaving(true)
    try {
      await updateJob(number, { note })
      setDraft(null)
      onSaved(note.trim() === '' ? `Anotação da vaga #${number} apagada.` : `Anotação da vaga #${number} salva.`)
    } catch (err) {
      onError(err instanceof Error ? err.message : 'Não consegui salvar a anotação.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <article className="detail" aria-labelledby="job-title">
      <BackLink href={backHref} />
      <header className="detail-head">
        <ScoreBadge score={job.score} large />
        <div>
          <h1 id="job-title">{job.title || 'Vaga sem título'}</h1>
          <p className="job-meta">
            <span className="job-company">{job.company || 'empresa não informada'}</span>
            <span className="job-number">#{job.number}</span>
            {(job.location || job.posting?.location) && <span>{job.location || job.posting?.location}</span>}
          </p>
          <p className="chips">
            <State job={job} />
            <Facts job={job} />
          </p>
        </div>
      </header>

      {job.gone && (
        <p className="notice">
          Esta vaga não veio mais na coleta: a empresa pode ter fechado a posição. Ela continua aqui com o que você marcou.
        </p>
      )}

      <section className="card" aria-labelledby="state-title">
        <h2 id="state-title">O que você fez com ela</h2>
        <StatusPicker number={job.number} status={job.status} busy={busy.has(job.number)} onMark={(to) => onMark(job, to)} />
        <p className="hint">
          Está como "{STATUS_LABEL[job.status]}" desde {formatDay(job.since)}. O Garimpo nunca se candidata por você: marcar "apliquei" é
          você contando que se candidatou.
        </p>
        {url ? (
          <a className="button button-primary external-button" href={url} target="_blank" rel="noopener noreferrer">
            Abrir a vaga no site<span aria-hidden="true"> ↗</span>
            <span className="visually-hidden"> (abre em outra aba)</span>
          </a>
        ) : (
          <p className="hint">Esta vaga veio sem um endereço válido.</p>
        )}
      </section>

      <section className="card" aria-labelledby="note-title">
        <h2 id="note-title">
          <label htmlFor="note">Anotação</label>
        </h2>
        <textarea
          id="note"
          rows={4}
          maxLength={MAX_NOTE}
          placeholder="Com quem falou, faixa salarial, link do processo…"
          value={note}
          onChange={(event) => setDraft(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'Enter' && (event.ctrlKey || event.metaKey)) void saveNote()
          }}
        />
        <div className="note-actions">
          <button type="button" className="button button-primary" disabled={!changed || saving} onClick={() => void saveNote()}>
            {saving ? 'Salvando…' : 'Salvar anotação'}
          </button>
          {changed && !saving && (
            <button type="button" className="button button-quiet" onClick={() => setDraft(null)}>
              Desfazer mudanças
            </button>
          )}
        </div>
      </section>

      <section className="card" aria-labelledby="score-title">
        <h2 id="score-title">Por que a nota é {job.score}</h2>
        {reasons.rules.length === 0 && !reasons.ai ? (
          <p className="hint">Os motivos aparecem depois da próxima coleta que listar esta vaga.</p>
        ) : (
          <ul className="reasons">
            {reasons.rules.map((reason, index) => (
              <li key={index}>
                <span className="points">{reason.points}</span>
                <span>{reason.text}</span>
              </li>
            ))}
          </ul>
        )}
        <p className="hint">A nota é a da última coleta ({formatDay(job.last_seen)}) e é refeita a cada rodada.</p>
      </section>

      {reasons.ai && (
        <section className="card" aria-labelledby="ai-title">
          <h2 id="ai-title">Leitura da IA</h2>
          {reasons.ai.fit === '' ? (
            <p className="hint">A IA não avaliou esta vaga{reasons.ai.summary ? `: ${reasons.ai.summary}` : ''}.</p>
          ) : (
            <>
              <p className="verdict">
                <span className="points">{reasons.ai.points}</span>
                <span>
                  Encaixe <strong>{reasons.ai.fit}</strong>
                </span>
              </p>
              <p>{reasons.ai.summary}</p>
              <VerdictList title="A favor" items={reasons.ai.strengths} />
              <VerdictList title="O que falta" items={reasons.ai.gaps} />
              <VerdictList title="Para destacar na candidatura" items={reasons.ai.highlights} />
            </>
          )}
        </section>
      )}

      <section className="card" aria-labelledby="history-title">
        <h2 id="history-title">Histórico</h2>
        <ol className="history">
          <li>
            <time dateTime={job.first_seen}>{formatMoment(job.first_seen)}</time>
            <span>o Garimpo achou a vaga</span>
          </li>
          {job.history.map((change, index) => (
            <li key={index}>
              <time dateTime={change.at}>{formatMoment(change.at)}</time>
              <span>
                {STATUS_LABEL[change.from]} → <strong>{STATUS_LABEL[change.to]}</strong>
              </span>
            </li>
          ))}
        </ol>
      </section>

      {job.posting && job.posting.description.trim() !== '' && (
        <section className="card" aria-labelledby="text-title">
          <h2 id="text-title">Texto da vaga</h2>
          <p className="hint">
            Fonte: {job.posting.source}
            {job.posting.posted_at && `, publicada em ${formatDay(job.posting.posted_at)}`}
          </p>
          {job.posting.tags.length > 0 && (
            <p className="chips">
              {job.posting.tags.map((tag) => (
                <span key={tag} className="chip">
                  {tag}
                </span>
              ))}
            </p>
          )}
          {/* Text from the internet: always shown as text, never as HTML. */}
          <div className="posting-text">{job.posting.description}</div>
        </section>
      )}
    </article>
  )
}

function BackLink({ href }: { href: string }) {
  return (
    <a className="back" href={href}>
      <span aria-hidden="true">← </span>Voltar para a lista
    </a>
  )
}

function VerdictList({ title, items }: { title: string; items: string[] }) {
  if (items.length === 0) return null
  return (
    <>
      <h3>{title}</h3>
      <ul className="verdict-list">
        {items.map((item, index) => (
          <li key={index}>{item}</li>
        ))}
      </ul>
    </>
  )
}
