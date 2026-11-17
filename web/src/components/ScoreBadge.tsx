/**
 * The score of the last run, as a nugget: the more gold, the better the
 * posting did. The number is always there; the fill only repeats it.
 */
export function ScoreBadge({ score, large = false }: { score: number; large?: boolean }) {
  const tier = score >= 70 ? 'high' : score >= 55 ? 'mid' : 'low'
  return (
    <span className={`score score-${tier}${large ? ' score-large' : ''}`} title="Nota da última coleta, de 0 a 100">
      <span className="visually-hidden">nota </span>
      {score}
    </span>
  )
}
