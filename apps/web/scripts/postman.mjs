// Generates the Postman collection from the OpenAPI spec and normalises it so the output is byte-stable.
// Rules: docs/design/P12-L1.md section 4.
import { readFileSync, realpathSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import converter from 'openapi-to-postmanv2'

const CAPTURE = [
  "const c = pm.response.headers.all().filter(h => h.key.toLowerCase() === 'set-cookie').map(h => h.value).find(v => v.startsWith('unicourse_session=') && !v.startsWith('unicourse_session=;'));",
  "pm.test('signed in (302 with a session cookie)', () => { pm.expect(pm.response.code).to.equal(302); pm.expect(c, 'no session cookie: token used, expired or wrong').to.be.a('string'); });",
  "if (c) pm.environment.set('session', c.split(';')[0].slice('unicourse_session='.length));",
]
const SEND = [
  "const s = pm.environment.get('session');",
  "if (s) pm.request.headers.upsert({ key: 'Cookie', value: 'unicourse_session=' + s });",
]
const script = (listen, exec) => ({ listen, script: { type: 'text/javascript', exec } })
const DROP = new Set(['id', '_postman_id', 'auth'])
const PATH_VALUES = { slug: '{{slug}}', token: '{{token}}' }

function strip(o) {
  if (Array.isArray(o)) return o.forEach(strip)
  if (o && typeof o === 'object') {
    for (const k of Object.keys(o)) DROP.has(k) ? delete o[k] : strip(o[k])
  }
}

function requests(items) {
  return items.flatMap((i) => (i.item ? requests(i.item) : i.request ? [i] : []))
}

function convert(specText) {
  const options = { folderStrategy: 'Paths', parametersResolution: 'Example', schemaFaker: false, includeAuthInfoInExample: false }
  return new Promise((resolve, reject) =>
    converter.convert({ type: 'string', data: specText }, options, (err, res) =>
      err || !res.result ? reject(new Error(`postman: ${err?.message ?? err ?? res.reason}`)) : resolve(res.output[0].data),
    ),
  )
}

export async function buildCollection(specText) {
  const collection = await convert(specText)
  strip(collection)
  const all = requests(collection.item)
  for (const i of all) {
    i.response = []
    for (const v of i.request.url?.variable ?? []) v.value = PATH_VALUES[v.key] ?? ''
    for (const q of i.request.url?.query ?? []) Object.assign(q, { value: '', disabled: true })
  }
  collection.variable = []
  const find = (path) => {
    const i = all.find((x) => x.request.method === 'POST' && x.request.url?.path?.join('/') === path.join('/'))
    if (!i) throw new Error(`postman: request ${path.join('/')} not found in the spec`)
    return i
  }
  find(['auth', 'magic-link']).request.body.raw = '{\n  "email": "{{email}}"\n}'
  const consume = find(['auth', 'magic-link', ':token'])
  consume.protocolProfileBehavior = { followRedirects: false }
  consume.event = [script('test', CAPTURE)]
  collection.event = [script('prerequest', SEND)]
  return collection
}

if (process.argv[1] && realpathSync(process.argv[1]) === realpathSync(fileURLToPath(import.meta.url))) {
  const [spec, out] = process.argv.slice(2)
  if (!out) {
    console.error('usage: postman.mjs <openapi.json> <out.json>')
    process.exit(2)
  }
  try {
    const c = await buildCollection(readFileSync(spec, 'utf8'))
    writeFileSync(out, JSON.stringify(c, null, 2) + '\n')
  } catch (e) {
    console.error(e.message.startsWith('postman:') ? e.message : `postman: ${e.message}`)
    process.exit(1)
  }
}
