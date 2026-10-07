import type { components } from '../api/types'

export type Meeting = components['schemas']['Meeting']
export type Slot = Pick<Meeting, 'weekday' | 'startsAt' | 'endsAt'>

export const dayShort = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'] as const
export const dayLong = ['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday'] as const

/** Same weekday and the half-open intervals [start, end) intersect. "HH:MM" compares as a string. */
export function overlaps(a: Slot, b: Slot): boolean {
  return a.weekday === b.weekday && a.startsAt < b.endsAt && b.startsAt < a.endsAt
}

/** Every other section (id !== section.id) with a meeting overlapping one of section.meetings, once each, in the order of `others`. */
export function clashesFor<S extends { id: string; meetings: Slot[] }>(
  section: { id: string; meetings: Slot[] },
  others: S[],
): S[] {
  return others.filter((o) => o.id !== section.id && o.meetings.some((m) => section.meetings.some((x) => overlaps(x, m))))
}

type TermLike = { startsOn: string; endsOn: string }

/** Term containing today (earliest start if several), else the soonest upcoming, else undefined. */
export function currentTerm<T extends TermLike>(terms: T[], today: string): T | undefined {
  const byStart = [...terms].sort((a, b) => a.startsOn.localeCompare(b.startsOn))
  return byStart.find((t) => t.startsOn <= today && today <= t.endsOn) ?? byStart.find((t) => t.startsOn > today)
}

export function classCode(s: { course: { code: string }; number: string }): string {
  return `${s.course.code}-${s.number}`
}

/** 'Mon 10:00–11:15 · Room 101' (long: 'Monday ...'); location omitted when empty. */
export function formatMeeting(m: Slot & { location: string }, long = false): string {
  const day = (long ? dayLong : dayShort)[m.weekday - 1]
  return `${day} ${m.startsAt}–${m.endsAt}${m.location ? ` · ${m.location}` : ''}`
}

export function toMinutes(t: string): number {
  const [h, m] = t.split(':')
  return Number(h) * 60 + Number(m)
}

/** Greedy lanes for slots sorted by startsAt then endsAt. */
export function assignLanes(slots: Pick<Slot, 'startsAt' | 'endsAt'>[]): { lane: number[]; lanes: number } {
  const lastEnd: string[] = []
  const lane = slots.map((s) => {
    let i = lastEnd.findIndex((e) => e <= s.startsAt)
    if (i < 0) i = lastEnd.length
    lastEnd[i] = s.endsAt
    return i
  })
  return { lane, lanes: Math.max(1, lastEnd.length) }
}

/** Meetings with the same time and room merged into one line each: 'Mon, Wed 10:00–11:20 · Hall B 204'. Lines ordered by their first weekday. */
export function formatMeetings(ms: (Slot & { location: string })[]): string[] {
  const groups = new Map<string, (Slot & { location: string })[]>()
  for (const m of [...ms].sort((a, b) => a.weekday - b.weekday || a.startsAt.localeCompare(b.startsAt))) {
    const k = `${m.startsAt}|${m.endsAt}|${m.location}`
    groups.set(k, [...(groups.get(k) ?? []), m])
  }
  return [...groups.values()].map((g) => {
    const { startsAt, endsAt, location } = g[0]
    return `${g.map((m) => dayShort[m.weekday - 1]).join(', ')} ${startsAt}–${endsAt}${location ? ` · ${location}` : ''}`
  })
}
