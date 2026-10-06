import { useMutation } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { api, unwrap } from "../../api/client";
import type { components } from "../../api/types";
import { Badge, RoleBadge, roleLabel } from "../../components/badge";
import { Button } from "../../components/button";
import { Dialog } from "../../components/dialog";
import { EmptyState } from "../../components/empty-state";
import { ErrorBanner } from "../../components/error-banner";
import {
  FormErrors,
  errorMessage,
  useFieldErrors,
  useFocusFirstError,
} from "../../components/form-errors";
import { FormField, inputClass } from "../../components/form-field";
import { LoadMore } from "../../components/load-more";
import { Loading } from "../../components/loading";
import { PageHeader } from "../../components/page-header";
import { StatusMessage } from "../../components/status-message";
import { Table } from "../../components/table";
import { useCursorList } from "../../lib/use-cursor-list";
import { useMe } from "../auth/use-me";
import { DepartmentPicker } from "./department-picker";
import { useAllDepartments, useRefresh } from "./queries";

type Membership = components["schemas"]["Membership"];
type Role = Membership["role"];

const roleCopy = {
  "validation_failed:departmentIds":
    "Choose at least one department for a department admin.",
  "conflict:role":
    "This is the university's only admin. Make someone else an admin first.",
  not_found: "That department no longer exists. Reload and try again.",
};
const listCopy = { forbidden: "You don't have permission to view this." };
const removeCopy = {
  conflict: "This is the university's only admin and can't be removed.",
};

function RoleForm(props: {
  slug: string;
  member: Membership;
  onDone: (notice?: string) => void;
  onCancel: () => void;
}) {
  const { slug, member, onDone, onCancel } = props;
  const refresh = useRefresh(slug, "memberships");
  const [role, setRole] = useState<Role>(member.role);
  const [departmentIds, setDepartmentIds] = useState(member.departmentIds);
  const formRef = useRef<HTMLFormElement>(null);
  const bannerRef = useRef<HTMLDivElement>(null);
  const m = useMutation({
    mutationFn: (body: { role: Role; departmentIds: string[] }) =>
      unwrap(
        api.PATCH("/universities/{slug}/memberships/{id}", {
          params: { path: { slug, id: member.id } },
          body,
        }),
      ),
    onSuccess: async (_, body) => {
      await refresh(["me"]);
      onDone(`${member.email} is now ${roleLabel(body.role)}.`);
    },
  });
  const fields = useFieldErrors(m.error, roleCopy);
  useFocusFirstError(formRef, m.error, bannerRef);

  return (
    <form
      ref={formRef}
      noValidate
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        m.mutate({
          role,
          departmentIds: role === "dept_admin" ? departmentIds : [],
        });
      }}
    >
      {Object.keys(fields).length === 0 && (
        <FormErrors ref={bannerRef} error={m.error} copy={roleCopy} />
      )}
      <FormField label="Role" name="role" error={fields.role}>
        <select
          className={inputClass}
          value={role}
          onChange={(e) => setRole(e.target.value as Role)}
        >
          <option value="admin">{roleLabel("admin")}</option>
          <option value="dept_admin">{roleLabel("dept_admin")}</option>
          <option value="member">{roleLabel("member")}</option>
        </select>
      </FormField>
      {role === "dept_admin" && (
        <DepartmentPicker
          slug={slug}
          value={departmentIds}
          onChange={setDepartmentIds}
          error={fields.departmentIds}
        />
      )}
      <div className="flex justify-end gap-2">
        <Button variant="secondary" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit" disabled={m.isPending}>
          {m.isPending ? "Saving…" : "Save role"}
        </Button>
      </div>
    </form>
  );
}

function RemoveForm(props: {
  slug: string;
  member: Membership;
  onDone: (notice?: string) => void;
  onCancel: () => void;
}) {
  const { slug, member, onDone, onCancel } = props;
  const refresh = useRefresh(slug, "memberships");
  const m = useMutation({
    mutationFn: () =>
      unwrap(
        api.DELETE("/universities/{slug}/memberships/{id}", {
          params: { path: { slug, id: member.id } },
        }),
      ),
    onSuccess: async () => {
      await refresh(["me"]);
      onDone(`${member.email} removed.`);
    },
  });
  return (
    <div className="flex flex-col gap-4">
      {m.error != null && (
        <ErrorBanner title="">{errorMessage(m.error, removeCopy)}</ErrorBanner>
      )}
      <div className="flex justify-end gap-2">
        <Button variant="secondary" onClick={onCancel}>
          Cancel
        </Button>
        <Button
          variant="danger"
          disabled={m.isPending}
          onClick={() => m.mutate()}
        >
          {m.isPending ? "Removing…" : "Remove"}
        </Button>
      </div>
    </div>
  );
}

export function MembersPage({ slug }: { slug: string }) {
  const list = useCursorList<Membership>([slug, "memberships"], (cursor) =>
    unwrap(
      api.GET("/universities/{slug}/memberships", {
        params: { path: { slug }, query: { cursor, limit: 50 } },
      }),
    ),
  );
  const depts = useAllDepartments(slug);
  const universityName =
    useMe().data?.memberships.find((x) => x.university.slug === slug)
      ?.university.name ?? "this university";
  const deptCode = new Map(depts.data?.items.map((d) => [d.id, d.code]));
  const [editing, setEditing] = useState<Membership | null>(null);
  const [removing, setRemoving] = useState<Membership | null>(null);
  const [notice, setNotice] = useState("");
  const done = (n?: string) => {
    setEditing(null);
    setRemoving(null);
    if (n) setNotice(n);
  };

  let body: React.ReactNode;
  if (list.isPending) body = <Loading />;
  else if (list.error)
    body = (
      <ErrorBanner onRetry={list.refetch}>
        {errorMessage(list.error, listCopy)}
      </ErrorBanner>
    );
  else if (list.items.length === 0)
    body = (
      <EmptyState
        title="No members yet"
        body="Invite people from Invitations."
      />
    );
  else
    body = (
      <>
        <Table caption="Members">
          <thead>
            <tr>
              <th scope="col">Name</th>
              <th scope="col">Role</th>
              <th scope="col">Departments</th>
              <th scope="col">Status</th>
              <th scope="col">
                <span className="sr-only">Actions</span>
              </th>
            </tr>
          </thead>
          <tbody>
            {list.items.map((x) => (
              <tr key={x.id}>
                <td>
                  {x.displayName || <span className="text-ink-muted">—</span>}
                  <div className="text-xs text-ink-muted">{x.email}</div>
                </td>
                <td>
                  <RoleBadge role={x.role} />
                </td>
                <td>
                  {x.departmentIds.length === 0
                    ? "—"
                    : x.departmentIds
                        .map((id) => deptCode.get(id) ?? "Unknown")
                        .join(", ")}
                </td>
                <td>
                  <Badge tone={x.status === "active" ? "neutral" : "warning"}>
                    {x.status === "active" ? "Active" : "Invited"}
                  </Badge>
                </td>
                <td className="text-right whitespace-nowrap">
                  <Button
                    variant="ghost"
                    size="sm"
                    aria-label={`Change role for ${x.email}`}
                    onClick={() => setEditing(x)}
                  >
                    Change role
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="text-danger hover:bg-danger-soft"
                    aria-label={`Remove ${x.email}`}
                    onClick={() => setRemoving(x)}
                  >
                    Remove
                  </Button>
                </td>
              </tr>
            ))}
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
      <PageHeader
        title="Members"
        description="People with access to this university. Invite new people from Invitations."
      />
      <StatusMessage message={notice} />
      {body}
      <Dialog
        open={editing !== null}
        onClose={() => setEditing(null)}
        title={`Change role for ${editing?.email ?? ""}`}
      >
        {editing && (
          <RoleForm
            key={editing.id}
            slug={slug}
            member={editing}
            onDone={done}
            onCancel={() => setEditing(null)}
          />
        )}
      </Dialog>
      <Dialog
        open={removing !== null}
        onClose={() => setRemoving(null)}
        title="Remove member?"
        description={
          removing
            ? `${removing.email} will lose access to ${universityName}. Their course staff assignments are not changed.`
            : undefined
        }
      >
        {removing && (
          <RemoveForm
            key={removing.id}
            slug={slug}
            member={removing}
            onDone={done}
            onCancel={() => setRemoving(null)}
          />
        )}
      </Dialog>
    </>
  );
}
