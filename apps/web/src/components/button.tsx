export const base =
  'inline-flex items-center justify-center gap-2 rounded font-bold whitespace-nowrap transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus disabled:cursor-not-allowed disabled:opacity-60'
export const sizes = { lg: 'h-12 px-5 text-base', md: 'h-10 px-4 text-sm', sm: 'h-8 px-3 text-sm' }
export const variants = {
  primary: 'bg-accent text-on-accent hover:bg-accent-hover',
  secondary: 'border border-line-control bg-surface text-ink hover:bg-line',
  danger: 'bg-danger text-on-accent hover:bg-danger-hover',
  ghost: 'text-accent hover:bg-accent-soft hover:text-accent-hover',
  'ghost-danger': 'text-ink-muted hover:bg-danger-soft hover:text-danger focus-visible:text-danger',
  'ghost-neutral': 'text-ink-muted hover:bg-canvas hover:text-ink',
}

export function Button(
  props: React.ButtonHTMLAttributes<HTMLButtonElement> & {
    variant?: keyof typeof variants
    size?: keyof typeof sizes
  },
) {
  const { className = '', type = 'button', variant = 'primary', size = 'md', ...rest } = props
  return <button type={type} className={`${base} ${sizes[size]} ${variants[variant]} ${className}`} {...rest} />
}
