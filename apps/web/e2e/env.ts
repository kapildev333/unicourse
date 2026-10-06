import path from 'node:path'
import { fileURLToPath } from 'node:url'

const here = path.dirname(fileURLToPath(import.meta.url))
export const tmpDir = path.join(here, '.tmp')
export const apiLog = path.join(tmpDir, 'api.log')
export const adminBin = path.join(tmpDir, 'unicourse-admin')
export const apiBin = path.join(tmpDir, 'unicourse-api')
export const webRoot = path.resolve(here, '..')
export const webDir = path.join(webRoot, 'dist')
export const apiDir = path.resolve(here, '..', '..', 'api')

export const databaseUrl =
  process.env.UNICOURSE_E2E_DATABASE_URL ?? 'postgres://unicourse:unicourse@localhost:5432/unicourse_e2e?sslmode=disable'

export const e2eEnv = {
  ...process.env,
  UNICOURSE_DATABASE_URL: databaseUrl,
  UNICOURSE_ADDR: '127.0.0.1:18180',
  UNICOURSE_BASE_URL: 'http://localhost:18180',
  UNICOURSE_SESSION_SECRET: 'e2e-session-secret-at-least-32-bytes-long',
  UNICOURSE_WEB_DIR: webDir,
  UNICOURSE_SMTP_URL: '',
}
