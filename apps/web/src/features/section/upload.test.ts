import { expect, it, vi } from 'vitest'
import { ApiError } from '../../api/errors'
import { finalLine, precheck, runUploads, uploadReason } from './upload'

const policy = { maxBytes: 104857600, allowedExtensions: ['pdf'] }
const file = (name: string, size = 5) => {
  const f = new File(['x'], name)
  Object.defineProperty(f, 'size', { value: size })
  return f
}

it('refuses bad files before sending', async () => {
  const send = vi.fn()
  const r = await runUploads([file('a.exe'), file('b.pdf', 0), file('c.pdf', 104857601)], policy, send, () => {})
  expect(send).not.toHaveBeenCalled()
  expect(r.failed.map((f) => f.reason)).toEqual([
    "This file type can't be uploaded.",
    'The file is empty.',
    'It is larger than 100 MB.',
  ])
})

it('sends in order and continues after a failure', async () => {
  const order: string[] = []
  let release!: () => void
  const gate = new Promise<void>((res) => (release = res))
  const lines: string[] = []
  const send = async (f: File) => {
    order.push(`start ${f.name}`)
    if (f.name === 'b.pdf') {
      await gate
      order.push('end b.pdf')
      throw new ApiError(400, 'validation_failed', 'm', [{ field: 'file', message: 'is empty' }])
    }
  }
  const p = runUploads([file('a.pdf'), file('b.pdf'), file('c.pdf')], policy, send, (l) => lines.push(l))
  await new Promise((r) => setTimeout(r, 10))
  expect(order).toEqual(['start a.pdf', 'start b.pdf'])
  release()
  const r = await p
  expect(order).toEqual(['start a.pdf', 'start b.pdf', 'end b.pdf', 'start c.pdf'])
  expect(lines).toEqual(['Uploading 1 of 3: a.pdf', 'Uploading 2 of 3: b.pdf', 'Uploading 3 of 3: c.pdf'])
  expect(r).toEqual({
    done: 2,
    total: 3,
    failed: [{ name: 'b.pdf', reason: 'The file is empty.' }],
  })
})

it('words upload reasons and final lines', () => {
  expect(uploadReason(new TypeError('Failed to fetch'))).toBe('The upload was interrupted. Check your connection and try again.')
  expect(uploadReason(new ApiError(400, 'validation_failed', 'm', [{ field: 'materials', message: 'x' }]))).toBe(
    'This week already has 200 files.',
  )
  expect(uploadReason(new ApiError(403, 'forbidden', 'm', []))).toContain('Only this section')
  expect(finalLine({ done: 1, total: 1 }, 'Week 1')).toBe('Uploaded 1 file to Week 1.')
  expect(finalLine({ done: 2, total: 3 }, 'Week 1')).toBe('Uploaded 2 of 3 files to Week 1.')
  expect(finalLine({ done: 0, total: 3 }, 'Week 1')).toBe('No files were uploaded to Week 1.')
})

it('prechecks against the policy, and not at all without one', () => {
  const p = { maxBytes: 10, allowedExtensions: ['pdf'] }
  expect(precheck(file('a.txt'), p)).toBe("This file type can't be uploaded.")
  expect(precheck(file('a.pdf', 11), p)).toBe('It is larger than 10 B.')
  expect(precheck(file('a.txt', 11))).toBeUndefined()
  expect(precheck(file('a.pdf', 0))).toBe('The file is empty.')
})

it('maps the server size text', () => {
  const e = new ApiError(400, 'validation_failed', 'm', [{ field: 'file', message: 'is larger than 1 MB' }])
  expect(uploadReason(e)).toBe('It is larger than 1 MB.')
})

it('shows whole-percent progress once per change', async () => {
  const lines: string[] = []
  await runUploads([file('a.pdf')], policy, async (_f, pct) => [0, 0, 45, 45, 100].forEach(pct), (l) => lines.push(l))
  expect(lines).toEqual([
    'Uploading 1 of 1: a.pdf',
    'Uploading 1 of 1: a.pdf (0%)',
    'Uploading 1 of 1: a.pdf (45%)',
    'Uploading 1 of 1: a.pdf (100%)',
  ])
})
