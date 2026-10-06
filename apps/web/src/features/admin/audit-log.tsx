import { useState } from "react";
import { api, unwrap } from "../../api/client";
import type { components } from "../../api/types";
import { Button } from "../../components/button";
import { EmptyState } from "../../components/empty-state";
import { ErrorBanner } from "../../components/error-banner";
import { errorMessage } from "../../components/form-errors";
import { FormField, inputClass } from "../../components/form-field";
import { LoadMore } from "../../components/load-more";
import { Loading } from "../../components/loading";
import { PageHeader } from "../../components/page-header";
import { Table } from "../../components/table";
import { formatDateTime } from "../../lib/format";
import { useCursorList } from "../../lib/use-cursor-list";
import { useAllMembers } from "./queries";

type Entry = components["schemas"]["AuditEntry"];
type EntityType = Entry["entityType"];

const entityLabels: Record<EntityType, string> = {
  department: "Department",
  term: "Term",
  membership: "Membership",
  invitation: "Invitation",
  course: "Course",
  section: "Section",
  staff: "Staff",
  enrollment: "Enrollment",
};
const verbs: Record<string, string> = {
  create: "created",
  update: "updated",
  delete: "deleted",
};
const listCopy = {
  validation_failed: "Those filters aren't valid. Clear them and try again.",
  forbidden: "You don't have permission to view this.",
};

type Filters = { entityType: string; actorUserId: string; from: string; to: string };
const none: Filters = { entityType: "", actorUserId: "", from: "", to: "" };

// Local midnight of a yyyy-mm-dd date, shifted by `addDays`, as an ISO instant.
function dayStart(date: string, addDays = 0): string | undefined {
  if (!date) return undefined;
  const [y, m, d] = date.split("-").map(Number);
  return new Date(y, m - 1, d + addDays).toISOString();
}

function formatValue(v: unknown): string {
  if (v === undefined || v === null || v === "") return "—";
  if (Array.isArray(v)) return v.length === 0 ? "—" : v.join(", ");
  if (typeof v === "object") return JSON.stringify(v);
  return String(v);
}

function Side(props: {
  title: string;
  data?: Record<string, unknown>;
  other?: Record<string, unknown>;
}) {
  const { title, data, other } = props;
  const keys = data ? Object.keys(data) : [];
  return (
    <div>
      <h3 className="text-xs font-semibold uppercase text-ink-muted">{title}</h3>
      {keys.length === 0 ? (
        <p className="text-xs">—</p>
      ) : (
        <dl>
          {keys.map((k) => {
            const changed =
              other !== undefined &&
              k in other &&
              JSON.stringify(other[k]) !== JSON.stringify(data![k]);
            return (
              <div key={k} className="flex gap-2 text-xs">
                <dt className="text-ink-muted">{k}</dt>
                <dd className={`font-mono break-all ${changed ? "font-semibold" : ""}`}>
                  {formatValue(data![k])}
                  {changed && " (changed)"}
                </dd>
              </div>
            );
          })}
        </dl>
      )}
    </div>
  );
}

export function AuditLogPage({ slug }: { slug: string }) {
  const [f, setF] = useState<Filters>(none);
  const members = useAllMembers(slug);
  const list = useCursorList<Entry>([slug, "audit-log", f], (cursor) =>
    unwrap(
      api.GET("/universities/{slug}/audit-log", {
        params: {
          path: { slug },
          query: {
            cursor,
            limit: 50,
            entityType: (f.entityType || undefined) as EntityType | undefined,
            actorUserId: f.actorUserId || undefined,
            from: dayStart(f.from),
            to: dayStart(f.to, 1),
          },
        },
      }),
    ),
  );
  const filtered = Object.values(f).some(Boolean);
  const set = (k: keyof Filters) => (e: React.ChangeEvent<HTMLElement & { value: string }>) =>
    setF({ ...f, [k]: e.target.value });
  const clear = (
    <Button variant="ghost" onClick={() => setF(none)}>
      Clear filters
    </Button>
  );

  let body: React.ReactNode;
  if (list.isPending) body = <Loading />;
  else if (list.error)
    body = (
      <ErrorBanner onRetry={list.refetch}>
        {errorMessage(list.error, listCopy)}
      </ErrorBanner>
    );
  else if (list.items.length === 0)
    body = filtered ? (
      <EmptyState title="No changes match these filters" body="Try a wider date range or fewer filters." action={clear} />
    ) : (
      <EmptyState title="No changes recorded yet" body="Changes made by admins will appear here." />
    );
  else
    body = (
      <>
        <Table caption="Audit log">
          <thead>
            <tr>
              <th scope="col">When</th>
              <th scope="col">Who</th>
              <th scope="col">Action</th>
              <th scope="col">Changes</th>
            </tr>
          </thead>
          <tbody>
            {list.items.map((e) => {
              const verb = e.action.split(".").pop() ?? e.action;
              return (
                <tr key={e.id}>
                  <td className="whitespace-nowrap">
                    <time dateTime={e.at}>{formatDateTime(e.at)}</time>
                  </td>
                  <td>{e.actor ? e.actor.displayName || e.actor.email : "System"}</td>
                  <td>
                    {entityLabels[e.entityType]} {verbs[verb] ?? verb}
                  </td>
                  <td>
                    <details>
                      <summary className="cursor-pointer text-sm text-accent">Show changes</summary>
                      <div className="mt-2 grid grid-cols-2 gap-4">
                        <Side title="Before" data={e.before} other={e.after} />
                        <Side title="After" data={e.after} other={e.before} />
                      </div>
                    </details>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </Table>
        <LoadMore
          hasNextPage={list.hasNextPage}
          isFetchingNextPage={list.isFetchingNextPage}
          onClick={list.fetchNextPage}
          shown={list.items.length}
        />
      </>
    );

  return (
    <>
      <PageHeader title="Audit log" description="Every change made in this university, newest first." />
      <form role="search" aria-label="Filter audit log" className="mb-4 flex flex-wrap items-end gap-3">
        <div className="w-48">
          <FormField label="Entity" name="entityType">
            <select className={inputClass} value={f.entityType} onChange={set("entityType")}>
              <option value="">All</option>
              {Object.entries(entityLabels).map(([v, l]) => (
                <option key={v} value={v}>
                  {l}
                </option>
              ))}
            </select>
          </FormField>
        </div>
        <div className="w-48">
          <FormField label="Changed by" name="actorUserId">
            <select className={inputClass} value={f.actorUserId} onChange={set("actorUserId")}>
              <option value="">Anyone</option>
              {members.data?.items.map((m) => (
                <option key={m.userId} value={m.userId}>
                  {m.email}
                </option>
              ))}
            </select>
          </FormField>
        </div>
        <div className="w-48">
          <FormField label="From" name="from">
            <input type="date" className={inputClass} value={f.from} onChange={set("from")} />
          </FormField>
        </div>
        <div className="w-48">
          <FormField label="To" name="to">
            <input type="date" className={inputClass} value={f.to} onChange={set("to")} />
          </FormField>
        </div>
        {filtered && clear}
      </form>
      {body}
    </>
  );
}
