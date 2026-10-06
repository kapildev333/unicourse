const dateOnly = /^(\d{4})-(\d{2})-(\d{2})$/

export function formatDate(s: string): string {
  const m = dateOnly.exec(s)
  // local date: new Date('YYYY-MM-DD') is UTC and shifts a day west of Greenwich
  const d = m ? new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3])) : new Date(s)
  return new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' }).format(d)
}

export function formatDateTime(iso: string): string {
  return new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(iso))
}
