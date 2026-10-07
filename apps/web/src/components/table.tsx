/** `stack`: below 640 px each row becomes a block; give every td and tbody th (row header) a `data-label` (CSS in index.css). */
export function Table(props: React.TableHTMLAttributes<HTMLTableElement> & { caption: string; stack?: boolean }) {
  const { caption, stack, children, className = '', ...rest } = props
  return (
    <div className="min-w-0">
      <div className="relative overflow-x-auto rounded border border-line bg-surface">
        <table className={`ui-table ${stack ? 'ui-table-stack' : ''} ${className}`} {...rest}>
          <caption className="sr-only">{caption}</caption>
          {children}
        </table>
      </div>
    </div>
  )
}
