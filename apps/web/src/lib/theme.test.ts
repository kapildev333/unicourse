/// <reference types="node" />
import { readFileSync } from 'node:fs'
import { beforeEach, expect, it } from 'vitest'
import { apply, boot, read, themeKey, write } from './theme'

// vitest blanks `?raw` imports of .css, so read the file directly (cwd is apps/web)
const css = readFileSync('src/index.css', 'utf8')
const root = document.documentElement
beforeEach(() => {
  localStorage.clear()
  root.removeAttribute('data-theme')
})

it('reads the stored choice and falls back to system', () => {
  expect(read()).toBe('system')
  write('dark')
  expect(read()).toBe('dark')
  localStorage.setItem(themeKey, 'blue')
  expect(read()).toBe('system')
})

it('stores system as no key', () => {
  write('dark')
  write('system')
  expect(localStorage.getItem(themeKey)).toBeNull()
})

it('apply sets or removes data-theme', () => {
  apply('dark')
  expect(root).toHaveAttribute('data-theme', 'dark')
  apply('light')
  expect(root).toHaveAttribute('data-theme', 'light')
  apply('system')
  expect(root).not.toHaveAttribute('data-theme')
})

it('boot applies the stored choice', () => {
  write('light')
  boot()
  expect(root).toHaveAttribute('data-theme', 'light')
})

it('the two dark blocks match', () => {
  const body = (re: RegExp) =>
    re.exec(css)![1].split(';').map((d) => d.trim()).filter(Boolean)
  const a = body(/html\[data-theme='dark'\]\s*\{([^}]*)\}/)
  const b = body(/html:not\(\[data-theme\]\)\s*\{([^}]*)\}/)
  expect(a).toHaveLength(19)
  expect(b).toEqual(a)
})
