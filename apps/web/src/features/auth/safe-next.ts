/** Returns `s` only when it is a same-origin path; never forward anything else as `next`. */
export function safeNext(s: unknown): string | undefined {
  return typeof s === 'string' &&
    s.startsWith('/') &&
    !s.startsWith('//') &&
    !s.startsWith('/\\') &&
    !/[\r\n\t]/.test(s) &&
    s.length <= 512
    ? s
    : undefined
}
