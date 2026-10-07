const url = /(https?:\/\/[^\s<>"]*[^\s<>".,:;!?)\]'])/g

/** Plain text with line breaks kept; http(s) URLs become links. Nothing else is interpreted. */
export function PlainText({ text, className = '' }: { text: string; className?: string }) {
  return (
    <div className={`whitespace-pre-wrap wrap-break-word ${className}`}>
      {text.split(url).map((part, i) =>
        i % 2 === 1 ? (
          <a key={i} href={part} target="_blank" rel="noopener noreferrer" className="text-accent underline hover:text-accent-hover">
            {part}
          </a>
        ) : (
          part
        ),
      )}
    </div>
  )
}

const n = (v: number) => v.toLocaleString('en-US')

/** "120 / 10,000 characters", or "10,012 / 10,000 characters, 12 too many". Code points, as the server counts. */
export function countHint(value: string, max: number): string {
  const c = [...value].length
  return `${n(c)} / ${n(max)} characters${c > max ? `, ${n(c - max)} too many` : ''}`
}
