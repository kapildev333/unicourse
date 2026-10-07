import { Link, type LinkProps } from '@tanstack/react-router'

/** Class for a link inside running text (also for a plain `<a>`): accent, underlined, darker on hover. */
export const textLink = 'text-accent underline hover:text-accent-hover'

export function TextLink(props: LinkProps & { className?: string; children?: React.ReactNode }) {
  const { className = '', ...rest } = props
  return <Link className={`${textLink} ${className}`.trim()} {...rest} />
}
