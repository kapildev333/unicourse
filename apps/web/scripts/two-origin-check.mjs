// Two-origin dev setup check (Spec 10 B11): API and Vite dev server on different origins, sign-in through the proxy.
// Env: WEB_URL (default http://localhost:5173), API_LOG (required, API log file), ADMIN_BIN (required, unicourse-admin;
// inherits UNICOURSE_* env), SLUG (default two-origin-<time>). Exit 1 with a message on any failure.
import { execFileSync } from 'node:child_process'
import fs from 'node:fs'

const web = (process.env.WEB_URL ?? 'http://localhost:5173').replace(/\/$/, '')
const { API_LOG, ADMIN_BIN } = process.env
const slug = process.env.SLUG ?? `two-origin-${Date.now().toString(36)}`
const email = `admin@${slug}.edu`

const fail = (msg) => {
  console.error(`two-origin-check FAILED: ${msg}`)
  process.exit(1)
}
const check = (ok, msg) => ok || fail(msg)
const step = (msg) => console.log(`ok: ${msg}`)

check(API_LOG && ADMIN_BIN, 'API_LOG and ADMIN_BIN must be set')

execFileSync(ADMIN_BIN, ['create-university', '--name', 'Two Origin', '--slug', slug, '--domains', `${slug}.edu`, '--timezone', 'UTC', '--admin-email', email], { stdio: 'pipe' })
step(`created university ${slug}`)

const req = await fetch(`${web}/api/v1/auth/magic-link`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json', 'Sec-Fetch-Site': 'same-origin' },
  body: JSON.stringify({ email }),
})
check(req.ok, `POST ${web}/api/v1/auth/magic-link through the proxy returned ${req.status}`)
step('magic link requested through the web origin')

// Dev mail is one JSON log line per message; poll for the sign-in link.
let link = ''
for (let i = 0; i < 40 && !link; i++) {
  for (const line of fs.readFileSync(API_LOG, 'utf8').split('\n')) {
    if (!line.includes('"msg":"mail"')) continue
    try {
      const m = JSON.parse(line)
      if (m.to === email) link = /https?:\/\/\S+\/api\/v1\/auth\/magic-link\/\S+/.exec(m.text)?.[0] ?? link
    } catch {
      // partial line
    }
  }
  if (!link) await new Promise((r) => setTimeout(r, 250))
}
check(link, `no sign-in mail for ${email} in ${API_LOG}`)
const u = new URL(link)
const path = u.pathname + u.search
step(`link read from the log (API origin ${u.origin})`)
check(u.origin !== web, `mail link origin ${u.origin} equals the web origin; the setup is not two-origin`)

const page = await fetch(web + path)
check(page.status === 200, `confirm page through the web origin returned ${page.status}`)
const csp = page.headers.get('content-security-policy') ?? ''
const formAction = /(?:^|;)\s*form-action ([^;]*)/.exec(csp)?.[1] ?? ''
check(formAction.split(/\s+/).includes(web), `CSP form-action "${formAction}" does not list the web origin ${web} (header: ${csp})`)
const html = await page.text()
check(new RegExp(`<form[^>]*action="${path.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}"`, 'i').test(html), `confirm page form does not post to ${path}`)
step(`confirm page CSP form-action lists ${web}; form posts to ${path}`)

const consume = await fetch(web + path, {
  method: 'POST',
  redirect: 'manual',
  headers: { 'Sec-Fetch-Site': 'same-origin', Origin: 'null', 'Content-Type': 'application/x-www-form-urlencoded' },
})
const loc = consume.headers.get('location')
check(consume.status === 302, `consume returned ${consume.status}, want 302`)
check(loc === `${web}/${slug}/me`, `consume redirected to ${loc}, want ${web}/${slug}/me`)
check((consume.headers.get('set-cookie') ?? '').includes('='), 'consume set no session cookie')
step(`consume redirected to ${loc}`)
console.log('two-origin-check passed')
