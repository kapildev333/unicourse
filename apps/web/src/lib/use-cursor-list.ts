import { useInfiniteQuery } from '@tanstack/react-query'

export function useCursorList<T>(
  queryKey: readonly unknown[],
  fetchPage: (cursor: string | undefined) => Promise<{ items: T[]; nextCursor?: string }>,
) {
  const q = useInfiniteQuery({
    queryKey,
    queryFn: ({ pageParam }) => fetchPage(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextCursor,
  })
  return {
    items: q.data?.pages.flatMap((p) => p.items) ?? [],
    isPending: q.isPending,
    error: q.error,
    refetch: () => void q.refetch(),
    hasNextPage: q.hasNextPage,
    fetchNextPage: () => void q.fetchNextPage(),
    isFetchingNextPage: q.isFetchingNextPage,
  }
}
