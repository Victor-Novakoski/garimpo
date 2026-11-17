import { describe, expect, it } from 'vitest'
import { formatDay, formatMoment, safeUrl, splitReasons } from './format'

describe('splitReasons', () => {
  it('separates the rules from the reading of the AI', () => {
    const got = splitReasons([
      '+15 Go no título',
      '+10 remoto (preferido)',
      '-5 inglês avançado',
      '+22 IA, encaixe bom: a maior parte bate: Go e React em produção',
      '    a favor: 3 anos de Go',
      '    a favor: PostgreSQL',
      '    falta: Kubernetes',
      '    destaque: a migração do monolito',
    ])
    expect(got.rules).toEqual([
      { points: '+15', text: 'Go no título' },
      { points: '+10', text: 'remoto (preferido)' },
      { points: '-5', text: 'inglês avançado' },
    ])
    expect(got.ai).toEqual({
      points: '+22',
      fit: 'bom',
      summary: 'a maior parte bate: Go e React em produção',
      strengths: ['3 anos de Go', 'PostgreSQL'],
      gaps: ['Kubernetes'],
      highlights: ['a migração do monolito'],
    })
  })

  it('says why the AI did not assess', () => {
    expect(splitReasons(['+0 IA não avaliou: tempo esgotado']).ai).toMatchObject({ fit: '', summary: 'tempo esgotado' })
    expect(splitReasons(['+0 IA não avaliou (fora das melhores, veja -ai-top)']).ai).toMatchObject({
      fit: '',
      summary: 'fora das melhores',
    })
  })

  it('has no AI part when the run had no AI', () => {
    expect(splitReasons(['+15 Go no título'])).toEqual({ rules: [{ points: '+15', text: 'Go no título' }], ai: null })
    expect(splitReasons([])).toEqual({ rules: [], ai: null })
  })

  it('keeps a line it does not know instead of hiding it', () => {
    const got = splitReasons(['uma linha nova do formato', '    a favor: sem IA antes'])
    expect(got.rules).toEqual([
      { points: '', text: 'uma linha nova do formato' },
      { points: '', text: 'a favor: sem IA antes' },
    ])
  })
})

describe('safeUrl', () => {
  it('only lets web addresses become links', () => {
    expect(safeUrl('https://empresa.com/vaga/1')).toBe('https://empresa.com/vaga/1')
    expect(safeUrl('http://empresa.com')).toBe('http://empresa.com/')
    expect(safeUrl('javascript:alert(1)')).toBeNull()
    expect(safeUrl('data:text/html,oi')).toBeNull()
    expect(safeUrl('')).toBeNull()
    expect(safeUrl('/caminho')).toBeNull()
  })
})

describe('dates', () => {
  it('formats as people write in Brazil', () => {
    const iso = new Date(2026, 10, 12, 14, 30).toISOString()
    expect(formatDay(iso)).toBe('12/11')
    expect(formatMoment(iso)).toBe('12/11/2026 às 14:30')
  })
  it('is empty for no date', () => {
    expect(formatDay(null)).toBe('')
    expect(formatMoment('nada')).toBe('')
  })
})
