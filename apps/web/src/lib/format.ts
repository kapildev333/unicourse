const dateOnly = /^(\d{4})-(\d{2})-(\d{2})$/

// The university's zone and locale, set once by routes/$slug/route.tsx; unset means the browser's.
let campus: { timeZone?: string; locale?: string } = {}

const valid = (o: Intl.DateTimeFormatOptions, locale?: string) => {
  try {
    new Intl.DateTimeFormat(locale, o)
    return true
  } catch {
    return false
  }
}

/** An invalid zone or locale falls back to the browser's. */
export function setCampus(u?: { timezone: string; locale: string }) {
  campus = {
    timeZone: u && valid({ timeZone: u.timezone }) ? u.timezone : undefined,
    locale: u && valid({}, u.locale) ? u.locale : undefined,
  }
}

export const browserZone = () => Intl.DateTimeFormat().resolvedOptions().timeZone
export const campusZone = () => campus.timeZone ?? browserZone()

export function formatDate(s: string): string {
  const m = dateOnly.exec(s)
  // a calendar date has no zone: format it as UTC so it never shifts a day
  if (m) {
    const d = new Date(Date.UTC(Number(m[1]), Number(m[2]) - 1, Number(m[3])))
    return new Intl.DateTimeFormat(campus.locale, { dateStyle: 'medium', timeZone: 'UTC' }).format(d)
  }
  return new Intl.DateTimeFormat(campus.locale, { dateStyle: 'medium', timeZone: campus.timeZone }).format(new Date(s))
}

/** dateStyle cannot be combined with timeZoneName, so the zone name comes from a second formatter. */
function zoneName(d: Date): string {
  const parts = new Intl.DateTimeFormat(campus.locale, { timeZone: campus.timeZone, timeZoneName: 'short' }).formatToParts(d)
  return parts.find((p) => p.type === 'timeZoneName')?.value ?? ''
}

export function formatDateTime(iso: string): string {
  const d = new Date(iso)
  const s = new Intl.DateTimeFormat(campus.locale, { dateStyle: 'medium', timeStyle: 'short', timeZone: campus.timeZone }).format(d)
  return campus.timeZone && campus.timeZone !== browserZone() ? `${s} ${zoneName(d)}` : s
}

type Fields = { y: number; mo: number; d: number; h: number; mi: number; wd: number }
const weekdays = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']

function fieldsIn(d: Date, timeZone: string): Fields {
  const parts = new Intl.DateTimeFormat('en-US', {
    timeZone,
    hourCycle: 'h23',
    year: 'numeric',
    month: 'numeric',
    day: 'numeric',
    hour: 'numeric',
    minute: 'numeric',
    weekday: 'short',
  }).formatToParts(d)
  const n = (t: string) => Number(parts.find((p) => p.type === t)?.value)
  const wd = weekdays.indexOf(parts.find((p) => p.type === 'weekday')?.value ?? '') + 1
  return { y: n('year'), mo: n('month'), d: n('day'), h: n('hour') % 24, mi: n('minute'), wd }
}

const pad = (n: number) => String(n).padStart(2, '0')
const isoDate = (f: Fields) => `${f.y}-${pad(f.mo)}-${pad(f.d)}`

/** The campus's calendar date, ISO weekday (1-7) and minutes since midnight. */
export function campusClock(d = new Date()): { date: string; weekday: number; minutes: number } {
  const f = fieldsIn(d, campusZone())
  return { date: isoDate(f), weekday: f.wd, minutes: f.h * 60 + f.mi }
}

/** Zone offset in ms at instant t: wall-clock-as-UTC minus t. */
function offset(t: number, zone: string): number {
  const f = fieldsIn(new Date(t), zone)
  return Date.UTC(f.y, f.mo - 1, f.d, f.h, f.mi) - Math.floor(t / 60_000) * 60_000
}

/** ISO instant -> `YYYY-MM-DDTHH:mm` in the campus zone, for datetime-local. */
export function toZonedInput(iso: string): string {
  const f = fieldsIn(new Date(iso), campusZone())
  return `${isoDate(f)}T${pad(f.h)}:${pad(f.mi)}`
}

/** datetime-local value (campus wall time) -> ISO instant; undefined for '' or invalid. */
export function fromZonedInput(v: string): string | undefined {
  const m = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})$/.exec(v)
  if (!m) return undefined
  const zone = campusZone()
  const guess = Date.UTC(+m[1], +m[2] - 1, +m[3], +m[4], +m[5])
  const first = guess - offset(guess, zone)
  const t = guess - offset(first, zone)
  return Number.isNaN(t) ? undefined : new Date(t).toISOString()
}
