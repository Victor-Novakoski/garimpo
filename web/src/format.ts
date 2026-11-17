import type { Status } from './types'

// The words of docs/FLOW.md, as the person reads them.
export const STATUS_LABEL: Record<Status, string> = {
  new: 'nova',
  saved: 'vou aplicar',
  applied: 'apliquei',
  interview: 'entrevista',
  discarded: 'descartada',
}

const WORK_MODE: Record<string, string> = { remote: 'remoto', hybrid: 'híbrido', onsite: 'presencial' }
const CONTRACT: Record<string, string> = { pj: 'PJ', clt: 'CLT', freelance: 'freelance', internship: 'estágio' }
const SENIORITY: Record<string, string> = { junior: 'júnior', mid: 'pleno', senior: 'sênior', lead: 'liderança' }

export const workModeLabel = (value: string): string => WORK_MODE[value] ?? value
export const contractLabel = (value: string): string => CONTRACT[value] ?? value
export const seniorityLabel = (value: string): string => SENIORITY[value] ?? value

const dayFormat = new Intl.DateTimeFormat('pt-BR', { day: '2-digit', month: '2-digit' })
const dayYearFormat = new Intl.DateTimeFormat('pt-BR', { day: '2-digit', month: '2-digit', year: 'numeric' })
const timeFormat = new Intl.DateTimeFormat('pt-BR', { hour: '2-digit', minute: '2-digit' })

function parse(iso: string | null | undefined): Date | null {
  if (!iso) return null
  const date = new Date(iso)
  return Number.isNaN(date.getTime()) ? null : date
}

/** "12/11" */
export function formatDay(iso: string | null | undefined): string {
  const date = parse(iso)
  return date ? dayFormat.format(date) : ''
}

/** "12/11/2026 às 14:30" */
export function formatMoment(iso: string | null | undefined): string {
  const date = parse(iso)
  return date ? `${dayYearFormat.format(date)} às ${timeFormat.format(date)}` : ''
}

/**
 * The address of a posting comes from the internet. Only a web address
 * becomes a link: anything else ("javascript:...") is dropped.
 */
export function safeUrl(url: string): string | null {
  try {
    const parsed = new URL(url)
    return parsed.protocol === 'https:' || parsed.protocol === 'http:' ? parsed.href : null
  } catch {
    return null
  }
}

export interface RuleReason {
  /** "+15", "-5"; empty when the line has no points. */
  points: string
  text: string
}

export interface AiVerdict {
  points: string
  /** "forte", "bom", "parcial", "fraco" or "nenhum"; empty when the AI did not assess. */
  fit: string
  summary: string
  strengths: string[]
  gaps: string[]
  highlights: string[]
}

export interface ScoreReasons {
  rules: RuleReason[]
  ai: AiVerdict | null
}

const POINTS = /^([+-]\d+)\s+(.*)$/
const AI_ASSESSED = /^IA, encaixe ([^:]+):\s*(.*)$/
const AI_SKIPPED = /^IA não avaliou:?\s*(.*)$/
const AI_DETAIL = /^\s+(a favor|falta|destaque):\s*(.*)$/

/**
 * Splits the reasons of a score into the rules and what the AI said.
 *
 * The API stores the reasons as the lines `garimpo collect -why` prints
 * (whyLines in cmd/garimpo/collect.go): one "+N text" per rule, then
 * "+N IA, encaixe <classe>: <resumo>" followed by indented "a favor:",
 * "falta:" and "destaque:" lines. A line this does not know stays as a rule
 * line, so nothing is hidden if the format grows.
 */
export function splitReasons(lines: readonly string[]): ScoreReasons {
  const out: ScoreReasons = { rules: [], ai: null }
  for (const line of lines) {
    const detail = AI_DETAIL.exec(line)
    if (detail && out.ai) {
      const text = detail[2] ?? ''
      if (detail[1] === 'a favor') out.ai.strengths.push(text)
      else if (detail[1] === 'falta') out.ai.gaps.push(text)
      else out.ai.highlights.push(text)
      continue
    }
    const scored = POINTS.exec(line.trim())
    const points = scored?.[1] ?? ''
    const text = scored?.[2] ?? line.trim()
    const assessed = AI_ASSESSED.exec(text)
    const skipped = AI_SKIPPED.exec(text)
    if (assessed) {
      out.ai = { points, fit: assessed[1] ?? '', summary: assessed[2] ?? '', strengths: [], gaps: [], highlights: [] }
    } else if (skipped) {
      const why = (skipped[1] ?? '').replace(/^\((.*)\)$/, '$1').replace(/, veja -ai-top$/, '')
      out.ai = { points, fit: '', summary: why, strengths: [], gaps: [], highlights: [] }
    } else if (text !== '') {
      out.rules.push({ points, text })
    }
  }
  return out
}
