export function Button(props: React.ButtonHTMLAttributes<HTMLButtonElement>) {
  const { className = '', type = 'button', ...rest } = props
  return (
    <button
      type={type}
      className={`rounded bg-blue-700 px-3 py-1.5 text-white hover:bg-blue-800 focus-visible:outline-2 focus-visible:outline-offset-2 disabled:opacity-50 ${className}`}
      {...rest}
    />
  )
}
