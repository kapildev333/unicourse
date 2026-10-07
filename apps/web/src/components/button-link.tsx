import { Link, type LinkProps } from '@tanstack/react-router'
import { base, sizes, variants } from './button'

/** A router link that looks exactly like `Button`; use it where a control navigates. */
export function ButtonLink(
  props: LinkProps & { variant?: keyof typeof variants; size?: keyof typeof sizes; className?: string; children?: React.ReactNode },
) {
  const { variant = 'primary', size = 'md', className = '', ...rest } = props
  return <Link className={`${base} ${sizes[size]} ${variants[variant]} ${className}`} {...rest} />
}
