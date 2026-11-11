import { describe, expect, it, vi } from 'vitest'
import { calculate } from '@/api/client'
import { Operation } from '@/api/constants'

/** A response like the backend's (or a proxy's) with a raw body. */
function respond(status: number, body: string, requestId = 'REQ-1'): Response {
  return new Response(body, { status, headers: { 'X-Request-ID': requestId } })
}

describe('calculate', () => {
  const cases = [
    {
      name: 'success',
      response: respond(200, '{"result":3.5}'),
      want: { kind: 'success', result: 3.5 },
    },
    {
      name: 'field errors are returned in order',
      response: respond(
        422,
        '[{"field":"leftOperand","code":"error.field.required"},{"field":"rightOperand","code":"error.field.invalid.type"}]',
      ),
      want: {
        kind: 'fieldErrors',
        errors: [
          { field: 'leftOperand', code: 'error.field.required' },
          { field: 'rightOperand', code: 'error.field.invalid.type' },
        ],
        requestId: 'REQ-1',
      },
    },
    {
      name: 'math error',
      response: respond(422, '{"code":"error.calculation.division.by.zero"}'),
      want: { kind: 'failure', code: 'error.calculation.division.by.zero', requestId: 'REQ-1' },
    },
    {
      name: 'operation not implemented on the server',
      response: respond(404, '{"code":"error.request.not.found"}'),
      want: { kind: 'failure', code: 'error.request.not.found', requestId: 'REQ-1' },
    },
    {
      name: 'internal error keeps the request id',
      response: respond(500, '{"code":"error.internal"}', 'ABC-123'),
      want: { kind: 'failure', code: 'error.internal', requestId: 'ABC-123' },
    },
    {
      name: 'unknown code is passed through',
      response: respond(422, '{"code":"error.calculation.new"}'),
      want: { kind: 'failure', code: 'error.calculation.new', requestId: 'REQ-1' },
    },
    {
      name: 'non-JSON error page from a proxy',
      response: respond(502, '<html>Bad Gateway</html>'),
      want: { kind: 'failure', code: 'error.client.unavailable', requestId: 'REQ-1' },
    },
    {
      name: 'empty 5xx body',
      response: respond(500, ''),
      want: { kind: 'failure', code: 'error.client.unavailable', requestId: 'REQ-1' },
    },
    {
      name: 'non-JSON 4xx',
      response: respond(400, 'bad request'),
      want: { kind: 'failure', code: 'error.client.unexpected.response', requestId: 'REQ-1' },
    },
    {
      name: 'success without a numeric result',
      response: respond(200, '{"result":"3"}'),
      want: { kind: 'failure', code: 'error.client.unexpected.response', requestId: 'REQ-1' },
    },
    {
      name: 'error body of an unexpected shape',
      response: respond(422, '{"message":"nope"}'),
      want: { kind: 'failure', code: 'error.client.unexpected.response', requestId: 'REQ-1' },
    },
    {
      name: 'empty field error array',
      response: respond(422, '[]'),
      want: { kind: 'failure', code: 'error.client.unexpected.response', requestId: 'REQ-1' },
    },
    {
      name: 'malformed field error entry',
      response: respond(422, '[{"field":1}]'),
      want: { kind: 'failure', code: 'error.client.unexpected.response', requestId: 'REQ-1' },
    },
    {
      name: 'response without a request id',
      response: new Response('{"code":"error.internal"}', { status: 500 }),
      want: { kind: 'failure', code: 'error.internal', requestId: null },
    },
  ]

  it.each(cases)('$name', async ({ response, want }) => {
    vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(response))
    expect(await calculate(Operation.Add, { leftOperand: 1, rightOperand: 2 })).toEqual(want)
  })

  it('reports a network failure', async () => {
    vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockRejectedValue(new TypeError('Failed to fetch')))

    expect(await calculate(Operation.Add, { leftOperand: 1, rightOperand: 2 })).toEqual({
      kind: 'failure',
      code: 'error.client.network',
      requestId: null,
    })
  })

  it('rethrows when the request is aborted while the body is read', async () => {
    const response = respond(200, '{"result":3}')
    vi.spyOn(response, 'json').mockRejectedValue(new DOMException('aborted', 'AbortError'))
    vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(response))

    await expect(calculate(Operation.Add, { leftOperand: 1, rightOperand: 2 })).rejects.toThrow('aborted')
  })

  it('rethrows when the request is aborted', async () => {
    vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockRejectedValue(new DOMException('aborted', 'AbortError')))

    await expect(calculate(Operation.Add, { leftOperand: 1, rightOperand: 2 })).rejects.toThrow('aborted')
  })

  it('posts JSON to the operation endpoint', async () => {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(respond(200, '{"result":3}'))
    vi.stubGlobal('fetch', fetchMock)
    const signal = new AbortController().signal

    await calculate(Operation.Divide, { leftOperand: 6, rightOperand: 2, precision: 1 }, signal)

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/divide', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      body: '{"leftOperand":6,"rightOperand":2,"precision":1}',
      signal,
    })
  })
})
