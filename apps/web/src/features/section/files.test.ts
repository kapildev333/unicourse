import { File as FileIcon, FileArchive, FileText, Image as ImageIcon, Presentation } from 'lucide-react'
import { expect, it } from 'vitest'
import { accept, extension, fileIcon, formatSize } from './files'

it('formats sizes', () => {
  const cases: [number, string][] = [
    [1, '1 B'],
    [1023, '1023 B'],
    [1024, '1 KB'],
    [1536, '1.5 KB'],
    [482113, '471 KB'],
    [1258291, '1.2 MB'],
    [10485760, '10 MB'],
    [104857600, '100 MB'],
  ]
  for (const [n, s] of cases) expect(formatSize(n)).toBe(s)
})

it('reads extensions and builds accept', () => {
  expect(extension('Deck.PDF')).toBe('pdf')
  expect(extension('deck.tar.gz')).toBe('gz')
  expect(extension('README')).toBe('')
  expect(accept({ maxBytes: 1, allowedExtensions: ['pdf', 'png'] })).toBe('.pdf,.png')
})

it('picks icons by type', () => {
  expect(fileIcon('application/pdf')).toBe(FileText)
  expect(fileIcon('application/vnd.openxmlformats-officedocument.wordprocessingml.document')).toBe(FileText)
  expect(fileIcon('text/plain; charset=utf-8')).toBe(FileText)
  expect(fileIcon('image/png')).toBe(ImageIcon)
  expect(fileIcon('application/zip')).toBe(FileArchive)
  expect(fileIcon('application/vnd.openxmlformats-officedocument.presentationml.presentation')).toBe(Presentation)
  expect(fileIcon('application/vnd.openxmlformats-officedocument.spreadsheetml.sheet')).toBe(FileIcon)
})
