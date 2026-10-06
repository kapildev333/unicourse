export function Table(props: React.TableHTMLAttributes<HTMLTableElement> & { caption: string }) {
  const { caption, children, className = '', ...rest } = props
  return (
    <table className={`w-full border-collapse text-left ${className}`} {...rest}>
      <caption className="sr-only">{caption}</caption>
      {children}
    </table>
  )
}
