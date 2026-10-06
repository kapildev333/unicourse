import fs from 'node:fs'
import { tmpDir } from './env'

export default function globalTeardown() {
  try {
    process.kill(Number(fs.readFileSync(`${tmpDir}/api.pid`, 'utf8')))
  } catch {
    // already gone
  }
}
