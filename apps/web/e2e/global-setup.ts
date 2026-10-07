import { execFileSync, spawn } from 'node:child_process'
import fs from 'node:fs'
import { adminBin, apiBin, apiDir, apiLog, e2eEnv, filesDir, tmpDir, webRoot } from './env'

// Builds the web app and both Go binaries, migrates, and starts the API serving the built web app.
export default async function globalSetup() {
  const base = 'http://localhost:18180'
  const taken = await fetch(base).then(() => true, () => false)
  if (taken) throw new Error(`something already answers on ${base}; stop it before running e2e`)
  fs.rmSync(tmpDir, { recursive: true, force: true })
  fs.mkdirSync(tmpDir, { recursive: true })
  fs.mkdirSync(filesDir)
  const run = (cmd: string, args: string[], cwd?: string) => execFileSync(cmd, args, { cwd, env: e2eEnv, stdio: 'inherit' })
  run('npm', ['run', 'build'], webRoot)
  run('go', ['build', '-o', apiBin, './cmd/api'], apiDir)
  run('go', ['build', '-o', adminBin, './cmd/unicourse-admin'], apiDir)
  run(adminBin, ['migrate'], apiDir)

  const log = fs.openSync(apiLog, 'a')
  const child = spawn(apiBin, [], { env: e2eEnv, stdio: ['ignore', log, log], detached: true })
  child.unref()
  fs.writeFileSync(`${tmpDir}/api.pid`, String(child.pid))

  const deadline = Date.now() + 30_000
  for (;;) {
    try {
      if ((await fetch(`${base}/sign-in`)).ok) return
    } catch {
      // not up yet
    }
    if (child.exitCode !== null) throw new Error(`API exited with code ${child.exitCode}; see ${apiLog}`)
    if (Date.now() > deadline) throw new Error(`API did not become ready; see ${apiLog}`)
    await new Promise((r) => setTimeout(r, 250))
  }
}
