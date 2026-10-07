/** Toolbar row above a list table: the loaded count, right-aligned (S5-34). */
export function ListCount({ shown }: { shown: number }) {
  return (
    <div className="mb-2 flex justify-end text-sm text-ink-muted">
      <span role="status">Showing {shown}</span>
    </div>
  );
}
