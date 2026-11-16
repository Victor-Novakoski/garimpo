import { useRef, useState } from 'react'
import { draftLetter } from '../api'

interface Props {
  number: number
  /** Why this server writes no letter; empty when it does. */
  off: string
  onCopied: (text: string) => void
  onError: (text: string) => void
}

/**
 * The draft of the cover letter. It lives only on this screen: the server
 * keeps nothing and sends nothing, and the person copies it to apply by
 * themselves. The text goes into a textarea, so it is never read as HTML.
 */
export function Letter({ number, off, onCopied, onError }: Props) {
  const [text, setText] = useState<string | null>(null)
  const [writing, setWriting] = useState(false)
  const [error, setError] = useState('')
  const field = useRef<HTMLTextAreaElement>(null)

  async function write() {
    if (writing) return
    setWriting(true)
    setError('')
    try {
      setText(await draftLetter(number))
    } catch (err) {
      // A draft already on the screen stays: it may have been edited.
      setError(err instanceof Error ? err.message : 'Não consegui gerar a carta.')
    } finally {
      setWriting(false)
    }
  }

  async function copy() {
    if (text === null) return
    try {
      await navigator.clipboard.writeText(text)
      onCopied('Carta copiada.')
    } catch {
      field.current?.select()
      onError('Não consegui copiar sozinho. O texto está selecionado: copie com Ctrl+C.')
    }
  }

  return (
    <section className="card" aria-labelledby="letter-title">
      <h2 id="letter-title">Carta de apresentação</h2>
      {text !== null && (
        <>
          <label className="visually-hidden" htmlFor="letter">
            Rascunho da carta
          </label>
          <textarea
            id="letter"
            ref={field}
            className="letter-text"
            rows={14}
            value={text}
            onChange={(event) => setText(event.target.value)}
          />
        </>
      )}
      {error && (
        <p className="notice notice-error" role="alert">
          {error}
        </p>
      )}
      <div className="note-actions">
        {text !== null && (
          <button type="button" className="button button-primary" disabled={text.trim() === ''} onClick={() => void copy()}>
            Copiar carta
          </button>
        )}
        <button
          type="button"
          className={text === null ? 'button button-primary' : 'button'}
          disabled={writing || off !== ''}
          onClick={() => void write()}
        >
          {writing ? 'Escrevendo a carta…' : text === null ? 'Gerar carta' : 'Gerar outra'}
        </button>
      </div>
      <p className="hint" aria-live="polite">
        {off !== ''
          ? `Sem carta por aqui: ${off}`
          : writing
            ? 'A IA está lendo a vaga e o seu perfil. Costuma levar menos de um minuto.'
            : text === null
              ? 'A IA escreve um rascunho a partir do seu perfil e do que ela apontou nesta vaga.'
              : 'É um rascunho: leia, ajuste aqui mesmo e envie você. O Garimpo não envia nem guarda a carta, e gerar outra troca este texto.'}
      </p>
    </section>
  )
}
