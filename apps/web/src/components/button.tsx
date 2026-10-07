const base =
  'inline-flex items-center justify-center gap-2 rounded-md font-medium whitespace-nowrap transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus disabled:cursor-not-allowed disabled:opacity-60'
const sizes = { md: 'h-9 px-4 text-sm', sm: 'h-8 px-3 text-sm' }
const variants = {
  primary: 'bg-accent text-on-accent hover:bg-accent-hover',
  secondary: 'border border-line-strong bg-surface text-ink hover:bg-canvas',
  danger: 'bg-danger text-on-accent hover:bg-danger-hover',
  ghost: 'text-accent hover:bg-accent-soft hover:text-accent-hover',
  'ghost-danger': 'text-danger hover:bg-danger-soft hover:text-danger-hover',
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
