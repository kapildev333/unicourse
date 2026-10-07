import { Table } from './table'

type Props = { label?: string } & (
  | { variant?: 'default' }
  | { variant: 'table'; columns: string[] }
  | { variant: 'cards' }
)

export function Loading(props: Props) {
  const label = props.label ?? 'Loading…'
  if (props.variant === 'table') {
    return (
      <div role="status" className="py-6">
        <span className="sr-only">{label}</span>
        <div aria-hidden="true">
          <Table caption={label}>
            <thead>
              <tr>
                {props.columns.map((c) => (
                  <th key={c}>{c}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {[0, 1, 2, 3, 4].map((r) => (
                <tr key={r}>
                  {props.columns.map((c, i) => (
                    <td key={c}>
                      <span className={`ui-skeleton ${i === 0 ? 'w-3/4' : 'w-1/2'}`} />
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </Table>
        </div>
      </div>
    )
  }
  if (props.variant === 'cards') {
    return (
      <div role="status" className="py-6">
        <span className="sr-only">{label}</span>
        <div aria-hidden="true" className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {[0, 1, 2].map((i) => (
            <div key={i} data-skeleton-card className="overflow-hidden rounded border border-line bg-surface">
              <div className="h-16 bg-accent-soft" />
              <div className="flex flex-col gap-2 p-4">
                <span className="ui-skeleton w-3/4" />
                <span className="ui-skeleton w-1/2" />
                <span className="ui-skeleton w-2/3" />
              </div>
            </div>
          ))}
        </div>
      </div>
    )
  }
  return (
    <div role="status" className="flex flex-col gap-3 py-6">
      <span className="sr-only">{label}</span>
      <span aria-hidden="true" className="ui-skeleton w-3/4" />
      <span aria-hidden="true" className="ui-skeleton w-full" />
      <span aria-hidden="true" className="ui-skeleton w-5/6" />
    </div>
  )
}
