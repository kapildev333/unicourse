import type { components } from '../../api/types'
import type { MyMembership } from '../auth/use-me'
import { sectionAccess } from '../courses/access'
import type { Assessment, GradeScaleRow } from './queries'

type Section = components['schemas']['Section']

export type Access = { view: boolean; scores: boolean; assessments: boolean; finalize: boolean }

/** Display only; the server authorises. Instructor: all; TA: scores; admin/dept_admin: view and finalize. */
export function gradebookAccess(m: MyMembership, userId: string, section: Section): Access {
  const { manager, staffRole } = sectionAccess(m, userId, section)
  const instructor = staffRole === 'instructor'
  return {
    view: manager || !!staffRole,
    scores: manager || !!staffRole,
    assessments: manager || instructor,
    finalize: instructor || manager,
  }
}

/** Half up to 2 decimals; toFixed(6) absorbs binary error (92.995*100 = 9299.4999…). */
export const round2 = (x: number) => Math.round(Number((x * 100).toFixed(6))) / 100
export const fmt2 = (x: number) => round2(x).toFixed(2)
export const fmt1 = (x: number) => x.toFixed(1)
export const fmtNum = (x: number) => String(Number(x.toFixed(2)))

const decimal = /^\d+(\.\d{1,2})?$/

/** '' or not a non-negative decimal with at most 2 places -> undefined. */
export function parseDecimal(text: string): number | undefined {
  const t = text.trim()
  return decimal.test(t) ? Number(t) : undefined
}

/** Parses a typed score. '' -> null (clear). */
export function parsePoints(text: string, maxPoints: number): { points: number | null } | { error: string } {
  const t = text.trim()
  if (t === '') return { points: null }
  const n = parseDecimal(t)
  if (n === undefined) return { error: 'Enter a number like 42 or 42.5.' }
  if (n > 2 * maxPoints) return { error: `Enter at most ${fmtNum(2 * maxPoints)} (twice the maximum).` }
  return { points: n }
}

/** Scale row with the greatest minPercent <= percent (unrounded); undefined below every row. */
export function letterFor(rows: GradeScaleRow[], percent: number): GradeScaleRow | undefined {
  let best: GradeScaleRow | undefined
  for (const r of rows) if (r.minPercent <= percent && (!best || r.minPercent > best.minPercent)) best = r
  return best
}

export const weightTotal = (as: Pick<Assessment, 'weight'>[]) => round2(as.reduce((s, a) => s + a.weight, 0))
export const totalsHundred = (as: Pick<Assessment, 'weight'>[]) => weightTotal(as) === 100

/** Finalization percent: missing scores count as 0; divisor is the sum of weights. */
export function finalPercent(as: Assessment[], scores: Record<string, number>): { percent: number; missing: number } {
  const total = as.reduce((s, a) => s + a.weight, 0)
  const earned = as.reduce((s, a) => s + (a.weight * (scores[a.id] ?? 0)) / a.maxPoints, 0)
  return { percent: total === 0 ? 0 : (earned / total) * 100, missing: as.filter((a) => scores[a.id] === undefined).length }
}

export type ScaleErrors = {
  rows: Record<number, Partial<Record<'letter' | 'minPercent' | 'points', string>>>
  table?: string
}

/** Client checks before PUT; no rows errors and no table error means valid. Rows are the editor's strings. */
export function validateScale(rows: { letter: string; minPercent: string; points: string }[]): ScaleErrors {
  const out: ScaleErrors = { rows: {} }
  const set = (i: number, f: 'letter' | 'minPercent' | 'points', msg: string) => {
    out.rows[i] = { ...out.rows[i], [f]: msg }
  }
  const letters = new Set<string>()
  const mins = new Set<number>()
  rows.forEach((r, i) => {
    const l = r.letter.trim()
    if (!l) set(i, 'letter', 'Enter a letter.')
    else if (letters.has(l)) set(i, 'letter', 'Each letter can appear only once.')
    letters.add(l)
    const p = parseDecimal(r.minPercent)
    if (p === undefined || p > 100) set(i, 'minPercent', 'Enter a percent from 0 to 100.')
    else {
      if (mins.has(p)) set(i, 'minPercent', 'Each minimum percent can appear only once.')
      mins.add(p)
    }
    const g = parseDecimal(r.points)
    if (g === undefined || g > 9.99) set(i, 'points', 'Enter grade points from 0 to 9.99.')
  })
  if (rows.length === 0) out.table = 'Add at least one row.'
  else if (!mins.has(0)) out.table = 'One row must start at 0 so every percent gets a letter.'
  return out
}

export const hasScaleErrors = (e: ScaleErrors) => !!e.table || Object.keys(e.rows).length > 0

export const sentence = (s: string) => {
  const t = s.trim()
  if (!t) return t
  const c = t[0].toUpperCase() + t.slice(1)
  return /[.!?]$/.test(c) ? c : c + '.'
}
