import { expect, it } from 'vitest'

const files = import.meta.glob<string>(['./**/*.{ts,tsx}', '!./**/*.test.{ts,tsx}'], { query: '?raw', import: 'default', eager: true })

const palette = 'slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose'
const rules: [string, RegExp][] = [
  ['hex colour', /(?<![\w&])#(?:[0-9a-fA-F]{3,4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})\b/],
  ['white/black utility', /\b[a-z-]+-(?:white|black)\b/],
  ['Tailwind palette utility', new RegExp(`\\b[a-z-]+-(?:${palette})-(?:50|[1-9]00|950)\\b`)],
  ['dark: variant', /(?:^|[\s'"`{])dark:/],
  ['colour function', /\b(?:rgba?|hsla?|oklch|oklab|lab|lch)\(/],
]

it('uses colour tokens only', () => {
  const hits: string[] = []
  for (const [file, src] of Object.entries(files))
    src.split('\n').forEach((line, i) => {
      for (const [name, re] of rules) if (re.test(line)) hits.push(`${file}:${i + 1} ${name}: ${line.trim()}`)
    })
  expect(hits).toEqual([])
})

it('scans the tree and every rule can fire', () => {
  expect(Object.keys(files).length).toBeGreaterThan(50)
  const bad = ["'#b54708'", 'bg-white', 'text-slate-500', '"dark:bg-surface"', 'rgb(0 0 0)']
  rules.forEach(([, re], i) => expect(re.test(bad[i])).toBe(true))
})
