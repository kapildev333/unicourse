export function Table(props: React.TableHTMLAttributes<HTMLTableElement> & { caption: string }) {
  const { caption, children, className = '', ...rest } = props
  return (
    <div className="overflow-x-auto rounded border border-line bg-surface">
      <table className={`ui-table ${className}`} {...rest}>
        <caption className="sr-only">{caption}</caption>
        {children}
      </table>
    </div>
  )
}
