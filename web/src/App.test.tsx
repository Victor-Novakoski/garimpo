import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { App } from './App'
import { fakeApi, job } from './test/fakeApi'

afterEach(() => {
  vi.unstubAllGlobals()
})

function open(hash = '') {
  window.location.hash = hash
  return render(<App pollMs={20} />)
}

const titles = () => screen.getAllByRole('heading', { level: 2 }).map((h) => h.textContent)

describe('the daily list', () => {
  it('shows what is left to decide, "vou aplicar" first, with score, company, model and link', async () => {
    fakeApi([
      job(1, { title: 'Dev Go', score: 50 }),
      job(2, { title: 'Dev React', score: 80, new_today: true, work_mode: '' }),
      job(3, { title: 'Dev Java', score: 30, status: 'saved', note: 'pede inglês' }),
      job(4, { title: 'Dev PHP', status: 'applied' }),
      job(5, { title: 'Dev Ruby', gone: true }),
    ])
    open()
    expect(await screen.findByRole('heading', { level: 2, name: 'Dev Java' })).toBeInTheDocument()
    expect(titles()).toEqual(['Dev Java', 'Dev React', 'Dev Go'])

    const saved = screen.getByRole('heading', { name: 'Dev Java' }).closest('li') as HTMLElement
    expect(within(saved).getByText('30')).toBeInTheDocument()
    expect(within(saved).getByText('Acme')).toBeInTheDocument()
    expect(within(saved).getByText('vou aplicar')).toBeInTheDocument()
    expect(within(saved).getByText('remoto')).toBeInTheDocument()
    expect(within(saved).getByText('pede inglês')).toBeInTheDocument()
    const link = within(saved).getByRole('link', { name: /Abrir a vaga/ })
    expect(link).toHaveAttribute('href', 'https://example.com/vagas/3')
    expect(link).toHaveAttribute('rel', 'noopener noreferrer')

    const fresh = screen.getByRole('heading', { name: 'Dev React' }).closest('li') as HTMLElement
    expect(within(fresh).getByText('nova hoje')).toBeInTheDocument()
    expect(within(fresh).getByText('modelo não informado')).toBeInTheDocument()
  })

  it('marks with one click, takes the posting off the list and lets the person undo', async () => {
    const api = fakeApi([job(1, { title: 'Dev Go' }), job(2, { title: 'Dev React' })])
    const user = userEvent.setup()
    open()
    const row = (await screen.findByRole('heading', { name: 'Dev Go' })).closest('li') as HTMLElement
    await user.click(within(row).getByRole('button', { name: 'Apliquei' }))

    await waitFor(() => expect(titles()).toEqual(['Dev React']))
    expect(api.calls).toContain('PATCH /api/jobs/1 {"status":"applied"}')
    expect(await screen.findByText('Vaga #1 marcada como "apliquei".')).toBeInTheDocument()
    // The counts of the filters follow.
    expect(within(screen.getByRole('link', { name: /^Apliquei/ })).getByText('1')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Desfazer' }))
    expect(await screen.findByRole('heading', { name: 'Dev Go' })).toBeInTheDocument()
    expect(api.calls).toContain('PATCH /api/jobs/1 {"status":"new"}')
    expect(api.jobs[0]?.status).toBe('new')
  })

  it('offers the usual next steps of each state', async () => {
    fakeApi([job(1, { status: 'saved' }), job(2)])
    open()
    const saved = (await screen.findByRole('heading', { name: 'Vaga 1' })).closest('li') as HTMLElement
    expect(within(saved).getAllByRole('button').map((b) => b.textContent)).toEqual(['Apliquei', 'Descartar'])
    const fresh = screen.getByRole('heading', { name: 'Vaga 2' }).closest('li') as HTMLElement
    expect(within(fresh).getAllByRole('button').map((b) => b.textContent)).toEqual(['Vou aplicar', 'Apliquei', 'Descartar'])
  })

  it('filters by state and by text', async () => {
    const api = fakeApi([
      job(1, { title: 'Dev Go', status: 'applied' }),
      job(2, { title: 'Dev React', status: 'applied', company: 'Beta' }),
      job(3, { title: 'Dev Java' }),
    ])
    const user = userEvent.setup()
    open()
    await screen.findByRole('heading', { name: 'Dev Java' })
    await user.click(screen.getByRole('link', { name: /^Apliquei/ }))
    expect(await screen.findByRole('heading', { level: 1, name: /Apliquei/ })).toBeInTheDocument()
    await waitFor(() => expect(titles()).toEqual(['Dev Go', 'Dev React']))
    expect(api.calls).toContain('GET /api/jobs?status=applied')
    expect(screen.getByRole('link', { name: /^Apliquei/ })).toHaveAttribute('aria-current', 'page')

    await user.type(screen.getByRole('searchbox'), 'beta')
    expect(titles()).toEqual(['Dev React'])
    await user.clear(screen.getByRole('searchbox'))
    await user.type(screen.getByRole('searchbox'), 'cobol')
    expect(screen.getByText(/Nenhuma vaga com "cobol"/)).toBeInTheDocument()
  })

  it('never turns an address that is not from the web into a link', async () => {
    fakeApi([job(1, { title: 'Dev Go', url: 'javascript:alert(1)' })])
    open()
    const row = (await screen.findByRole('heading', { name: 'Dev Go' })).closest('li') as HTMLElement
    expect(within(row).queryByRole('link', { name: /Abrir a vaga/ })).not.toBeInTheDocument()
  })

  it('says what went wrong when the server refuses', async () => {
    const api = fakeApi([job(1)])
    api.failNext = { status: 500, error: 'erro interno: veja o log do servidor' }
    open()
    expect(await screen.findByRole('alert')).toHaveTextContent('erro interno: veja o log do servidor')
  })

  it('says so when there is nothing to decide', async () => {
    fakeApi([])
    open()
    expect(await screen.findByText(/Nada para decidir agora/)).toBeInTheDocument()
    expect(screen.getByText('Nenhuma coleta ainda')).toBeInTheDocument()
  })
})

describe('the posting', () => {
  const full = () =>
    job(12, {
      title: 'Dev Go Pleno',
      score: 84,
      status: 'saved',
      note: 'falei com a Ana',
      reasons: [
        '+15 Go no título',
        '+10 remoto (preferido)',
        '+22 IA, encaixe bom: a maior parte bate',
        '    a favor: 3 anos de Go',
        '    falta: Kafka',
        '    destaque: a API de rastreio',
      ],
      history: [{ at: '2026-11-11T12:00:00Z', from: 'new', to: 'saved' }],
      posting: {
        source: 'greenhouse:acme',
        location: 'Brasil',
        description: 'Buscamos uma pessoa <b>dev</b>',
        tags: ['go'],
        work_mode: 'remote',
        contract: 'pj',
        seniority: 'mid',
        posted_at: null,
      },
    })

  it('shows the reasons of the score, the reading of the AI, the note and the history', async () => {
    fakeApi([full()])
    open('#/vaga/12')
    expect(await screen.findByRole('heading', { level: 1, name: 'Dev Go Pleno' })).toBeInTheDocument()

    const score = screen.getByRole('heading', { name: 'Por que a nota é 84' }).closest('section') as HTMLElement
    expect(within(score).getByText('Go no título')).toBeInTheDocument()
    expect(within(score).getByText('+15')).toBeInTheDocument()
    expect(within(score).queryByText(/IA/)).not.toBeInTheDocument()

    const ai = screen.getByRole('heading', { name: 'Leitura da IA' }).closest('section') as HTMLElement
    expect(within(ai).getByText('+22')).toBeInTheDocument()
    expect(within(ai).getByText('bom')).toBeInTheDocument()
    expect(within(ai).getByText('a maior parte bate')).toBeInTheDocument()
    expect(within(ai).getByText('3 anos de Go')).toBeInTheDocument()
    expect(within(ai).getByText('Kafka')).toBeInTheDocument()
    expect(within(ai).getByText('a API de rastreio')).toBeInTheDocument()

    expect(screen.getByRole('textbox', { name: 'Anotação' })).toHaveValue('falei com a Ana')
    const history = screen.getByRole('heading', { name: 'Histórico' }).closest('section') as HTMLElement
    expect(within(history).getByText('o Garimpo achou a vaga')).toBeInTheDocument()
    expect(within(history).getAllByRole('listitem')[1]).toHaveTextContent('nova → vou aplicar')

    // The text of the posting is text: its markup is never run.
    expect(screen.getByText('Buscamos uma pessoa <b>dev</b>')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Abrir a vaga no site/ })).toHaveAttribute('href', 'https://example.com/vagas/12')
  })

  it('marks any state from the posting', async () => {
    const api = fakeApi([full()])
    const user = userEvent.setup()
    open('#/vaga/12')
    const picker = await screen.findByRole('group', { name: 'Estado da vaga 12' })
    expect(within(picker).getByRole('button', { name: 'vou aplicar' })).toHaveAttribute('aria-pressed', 'true')
    await user.click(within(picker).getByRole('button', { name: 'entrevista' }))
    await waitFor(() => expect(within(picker).getByRole('button', { name: 'entrevista' })).toHaveAttribute('aria-pressed', 'true'))
    expect(api.calls).toContain('PATCH /api/jobs/12 {"status":"interview"}')
    expect(screen.getByRole('heading', { name: 'Histórico' }).closest('section')).toHaveTextContent('vou aplicar → entrevista')
  })

  it('saves and clears the note', async () => {
    const api = fakeApi([full()])
    const user = userEvent.setup()
    open('#/vaga/12')
    const note = await screen.findByRole('textbox', { name: 'Anotação' })
    const save = screen.getByRole('button', { name: 'Salvar anotação' })
    expect(save).toBeDisabled()

    await user.clear(note)
    await user.type(note, 'faixa de 9 a 11 mil')
    await user.click(save)
    expect(await screen.findByText('Anotação da vaga #12 salva.')).toBeInTheDocument()
    expect(api.jobs[0]?.note).toBe('faixa de 9 a 11 mil')
    await waitFor(() => expect(save).toBeDisabled())

    await user.clear(note)
    await user.click(save)
    expect(await screen.findByText('Anotação da vaga #12 apagada.')).toBeInTheDocument()
    expect(api.calls).toContain('PATCH /api/jobs/12 {"note":""}')
  })

  it('keeps what was typed when saving fails', async () => {
    const api = fakeApi([full()])
    const user = userEvent.setup()
    open('#/vaga/12')
    const note = await screen.findByRole('textbox', { name: 'Anotação' })
    await user.type(note, ' e mais')
    api.failNext = { status: 400, error: 'anotação longa demais: o limite é de 4000 caracteres' }
    await user.click(screen.getByRole('button', { name: 'Salvar anotação' }))
    expect(await screen.findByText(/anotação longa demais/)).toBeInTheDocument()
    expect(note).toHaveValue('falei com a Ana e mais')
  })

  it('says when the posting does not exist', async () => {
    fakeApi([])
    open('#/vaga/99')
    expect(await screen.findByRole('alert')).toHaveTextContent('não existe vaga #99')
  })
})

describe('the funnel', () => {
  it('counts each state and lists the changes, newest first', async () => {
    fakeApi([
      job(1, { title: 'Dev Go', status: 'interview', history: [
        { at: '2026-11-10T12:00:00Z', from: 'new', to: 'applied' },
        { at: '2026-11-12T12:00:00Z', from: 'applied', to: 'interview' },
      ] }),
      job(2, { status: 'discarded', gone: true, history: [{ at: '2026-11-11T12:00:00Z', from: 'new', to: 'discarded' }] }),
      job(3),
      job(4),
    ])
    open('#/funil')
    expect(await screen.findByText(/4 vagas no registro/)).toBeInTheDocument()
    const stages = within(screen.getAllByRole('list')[0] as HTMLElement).getAllByRole('link')
    expect(stages.map((s) => s.textContent)).toEqual([
      'nova2 vagas',
      'vou aplicar0 vagas',
      'apliquei0 vagas',
      'entrevista1 vaga',
      'descartada1 vaga1 saiu da fonte',
    ])
    expect(stages[3]).toHaveAttribute('href', '#/vagas/entrevista')

    const changes = within(screen.getAllByRole('list')[1] as HTMLElement).getAllByRole('listitem')
    expect(changes).toHaveLength(3)
    expect(changes[0]).toHaveTextContent('#1 Dev Go')
    expect(changes[0]).toHaveTextContent('apliquei → entrevista')
    expect(changes[2]).toHaveTextContent('nova → apliquei')
  })
})

describe('collecting from the panel', () => {
  it('starts a collection, follows it and loads the new postings', async () => {
    const api = fakeApi([job(1, { title: 'Dev Go' })])
    api.onCollect = (fake) => {
      fake.collect = { ...fake.collect, log: 'greenhouse:acme 12 vagas\n' }
      setTimeout(() => {
        fake.jobs.push(job(2, { title: 'Dev React', new_today: true, score: 90 }))
        fake.collect = { ...fake.collect, running: false, finished_at: '2026-11-12T13:05:00Z', log: 'greenhouse:acme 12 vagas\nlista: 2 vagas\n' }
      }, 60)
    }
    const user = userEvent.setup()
    open()
    await screen.findByRole('heading', { name: 'Dev Go' })
    await user.click(screen.getByRole('button', { name: 'Buscar vagas' }))

    expect(await screen.findByRole('button', { name: 'Buscando vagas…' })).toBeDisabled()
    expect(await screen.findByText(/greenhouse:acme 12 vagas/)).toBeInTheDocument()
    expect(api.calls).toContain('POST /api/collect')

    expect(await screen.findByText(/Coleta concluída/)).toBeInTheDocument()
    expect(await screen.findByRole('heading', { name: 'Dev React' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Buscar vagas' })).toBeEnabled()
  })

  it('shows why a collection failed', async () => {
    const api = fakeApi([])
    api.onCollect = (fake) => {
      fake.collect = { ...fake.collect, running: false, finished_at: '2026-11-12T13:05:00Z', error: 'nenhuma fonte respondeu: confira a internet e tente de novo' }
    }
    const user = userEvent.setup()
    open()
    await user.click(await screen.findByRole('button', { name: 'Buscar vagas' }))
    expect(await screen.findByText('A coleta não terminou')).toBeInTheDocument()
    expect(screen.getByText(/nenhuma fonte respondeu/)).toBeInTheDocument()
  })

  it('turns the button off when this server cannot collect', async () => {
    const api = fakeApi([])
    api.collect = { available: false, reason: 'a coleta é feita pelo terminal', running: false, started_at: null, finished_at: null, log: '' }
    open()
    await waitFor(() => expect(screen.getByRole('button', { name: 'Buscar vagas' })).toBeDisabled())
  })
})

describe('addresses', () => {
  it('says when a screen does not exist', async () => {
    fakeApi([])
    open('#/nada')
    expect(await screen.findByRole('heading', { name: 'Esta tela não existe' })).toBeInTheDocument()
  })
})
