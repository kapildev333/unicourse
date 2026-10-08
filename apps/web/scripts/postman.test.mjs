import { describe, expect, it } from 'vitest'
import { buildCollection, oneKey } from './postman.mjs'

const spec = (withConsume = true, withRequest = true) => ({
  openapi: '3.1.0',
  info: { title: 't', version: '1' },
  servers: [{ url: '/api/v1' }],
  security: [{ session: [] }],
  components: { securitySchemes: { session: { type: 'apiKey', in: 'cookie', name: 'unicourse_session' } } },
  paths: {
    ...(withRequest ? { '/auth/magic-link': {
      post: {
        summary: 'Request a magic link',
        requestBody: {
          content: { 'application/json': { schema: { type: 'object', properties: { email: { type: 'string' } } }, example: { email: 'a@b.c' } } },
        },
        responses: { 202: { description: 'ok', content: { 'application/json': { example: { ok: true } } } } },
      },
    } } : {}),
    '/things/{id}': {
      get: {
        summary: 'Get a thing',
        parameters: [{ name: 'id', in: 'path', required: true, schema: { type: 'string' } }],
        responses: { 200: { description: 'ok' } },
      },
    },
    ...(withConsume
      ? {
          '/auth/magic-link/{token}': {
            post: {
              summary: 'Consume a magic link',
              parameters: [{ name: 'token', in: 'path', required: true, schema: { type: 'string' } }],
              responses: { 302: { description: 'redirect' } },
            },
          },
        }
      : {}),
    '/universities/{slug}/sections': {
      get: {
        summary: 'List sections',
        parameters: [
          { name: 'slug', in: 'path', required: true, schema: { type: 'string' } },
          { name: 'termId', in: 'query', schema: { type: 'string', format: 'uuid' } },
        ],
        responses: { 200: { description: 'ok', content: { 'application/json': { example: [] } } } },
      },
    },
  },
})

const items = (c) => c.item.flatMap(function walk(i) { return i.item ? i.item.flatMap(walk) : [i] })
const find = (c, method, path) =>
  items(c).find((i) => i.request.method === method && i.request.url.path.join('/') === path)
const keys = (o, out = new Set()) => {
  if (o && typeof o === 'object') for (const [k, v] of Object.entries(o)) { out.add(k); keys(v, out) }
  return out
}

describe('buildCollection', () => {
  const text = JSON.stringify(spec())

  it('strips id, _postman_id and auth at any depth', async () => {
    const k = keys(await buildCollection(text))
    for (const bad of ['id', '_postman_id', 'auth']) expect(k.has(bad)).toBe(false)
  })

  it('empties responses and collection variables', async () => {
    const c = await buildCollection(text)
    expect(items(c).every((i) => Array.isArray(i.response) && i.response.length === 0)).toBe(true)
    expect(c.variable).toEqual([])
  })

  it('maps path and query variables', async () => {
    const c = await buildCollection(text)
    const r = find(c, 'GET', 'universities/:slug/sections').request
    expect(r.url.variable[0].value).toBe('{{slug}}')
    expect(r.url.query[0]).toMatchObject({ key: 'termId', value: '', disabled: true })
  })

  it('blanks other path variables', async () => {
    const c = await buildCollection(text)
    expect(find(c, 'GET', 'things/:id').request.url.variable[0].value).toBe('')
  })

  it('sets the magic-link body to the email variable', async () => {
    const c = await buildCollection(text)
    expect(JSON.parse(find(c, 'POST', 'auth/magic-link').request.body.raw)).toEqual({ email: '{{email}}' })
  })

  it('configures the consume item and the collection send script', async () => {
    const c = await buildCollection(text)
    const i = find(c, 'POST', 'auth/magic-link/:token')
    expect(i.protocolProfileBehavior).toEqual({ followRedirects: false })
    expect(i.request.url.variable[0].value).toBe('{{token}}')
    expect(i.event[0].listen).toBe('test')
    expect(i.event[0].script.exec.join('\n')).toContain("pm.environment.set('session'")
    expect(c.event[0].listen).toBe('prerequest')
    expect(c.event[0].script.exec.join('\n')).toContain("unicourse_session=")
  })

  it('is deterministic', async () => {
    expect(JSON.stringify(await buildCollection(text))).toBe(JSON.stringify(await buildCollection(text)))
  })

  it('rejects when the consume route is missing', async () => {
    await expect(buildCollection(JSON.stringify(spec(false)))).rejects.toThrow(/not found in the spec/)
  })

  it('rejects when the request route is missing', async () => {
    await expect(buildCollection(JSON.stringify(spec(true, false)))).rejects.toThrow(/not found in the spec/)
  })
})

describe('oneKey', () => {
  it('keeps only key_0 of a generated map, at any depth', () => {
    const got = oneKey({ answers: { key_0: { a: 1 }, key_1: { a: 1 }, key_2: { a: 1 } }, list: [{ key_0: 1, key_1: 2 }], submit: true })
    expect(got).toEqual({ answers: { key_0: { a: 1 } }, list: [{ key_0: 1 }], submit: true })
  })
  it('leaves ordinary objects alone', () => {
    expect(oneKey({ key_0: 1, other: 2 })).toEqual({ key_0: 1, other: 2 })
  })
})
