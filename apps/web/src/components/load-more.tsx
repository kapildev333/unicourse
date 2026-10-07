import { Button } from './button'

export function LoadMore(props: { hasNextPage: boolean; isFetchingNextPage: boolean; onClick: () => void; shown: number; showCount?: boolean }) {
  return (
    <div className="mt-4 flex items-center justify-between text-sm text-ink-muted">
      {props.showCount !== false ? <span role="status">Showing {props.shown}</span> : <span />}
      {props.hasNextPage && (
        <Button variant="secondary" size="sm" disabled={props.isFetchingNextPage} onClick={props.onClick}>
          {props.isFetchingNextPage ? 'Loading…' : 'Load more'}
        </Button>
      )}
    </div>
  )
}
