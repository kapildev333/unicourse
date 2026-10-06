import { Button } from './button'

export function LoadMore(props: { hasNextPage: boolean; isFetchingNextPage: boolean; onClick: () => void; shown: number }) {
  return (
    <div className="mt-4 flex items-center justify-between text-sm text-ink-muted">
      <span role="status">Showing {props.shown}</span>
      {props.hasNextPage && (
        <Button variant="secondary" size="sm" disabled={props.isFetchingNextPage} onClick={props.onClick}>
          {props.isFetchingNextPage ? 'Loading…' : 'Load more'}
        </Button>
      )}
    </div>
  )
}
