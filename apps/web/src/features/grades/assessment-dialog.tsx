import { useMutation } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { api, unwrap } from "../../api/client";
import { Button } from "../../components/button";
import { Dialog } from "../../components/dialog";
import {
  FormErrors,
  useFieldErrors,
  useFocusFirstError,
} from "../../components/form-errors";
import { FormField, inputClass } from "../../components/form-field";
import { fmtNum, parseDecimal } from "./grade";
import { useGradesRefresh, type Assessment } from "./queries";
import type { components } from "../../api/types";

const copy = {
  "validation_failed:name": "Enter a name of at most 100 characters.",
  "conflict:name": "This section already has an assessment with this name.",
  "validation_failed:maxPoints": "Enter maximum points greater than 0.",
  "validation_failed:weight": "Enter a weight from 0 to 100.",
  "validation_failed:dueOn": "Enter a valid date.",
  forbidden: "Only this section's instructors can change assessments.",
};

/** Mount only while open (parent renders it conditionally) so the form starts fresh each time. */
export function AssessmentDialog(props: {
  slug: string;
  sectionId: string;
  assessment?: Assessment;
  scoredCount: number;
  onClose: () => void;
  onSaved: (message: string) => void;
}) {
  const { slug, sectionId, assessment, scoredCount, onClose, onSaved } = props;
  const refresh = useGradesRefresh(slug);
  const [name, setName] = useState(assessment?.name ?? "");
  const [maxPoints, setMax] = useState(
    assessment ? fmtNum(assessment.maxPoints) : "",
  );
  const [weight, setWeight] = useState(
    assessment ? fmtNum(assessment.weight) : "",
  );
  const [dueOn, setDueOn] = useState(assessment?.dueOn ?? "");
  const [local, setLocal] = useState<Record<string, string>>({});
  const [confirmDelete, setConfirmDelete] = useState(false);
  const formRef = useRef<HTMLFormElement>(null);
  const bannerRef = useRef<HTMLDivElement>(null);
  const path = { params: { path: { slug, id: sectionId } } };

  const m = useMutation({
    mutationFn: (
      body:
        | components["schemas"]["AssessmentCreate"]
        | components["schemas"]["AssessmentUpdate"],
    ) =>
      assessment
        ? unwrap(
            api.PATCH(
              "/universities/{slug}/sections/{id}/assessments/{assessmentId}",
              {
                params: {
                  path: { slug, id: sectionId, assessmentId: assessment.id },
                },
                body,
              },
            ),
          )
        : unwrap(
            api.POST("/universities/{slug}/sections/{id}/assessments", {
              ...path,
              body: body as components["schemas"]["AssessmentCreate"],
            }),
          ),
    onSuccess: async (a) => {
      await refresh();
      onSaved(`${a.name} ${assessment ? "saved" : "created"}.`);
    },
  });
  const del = useMutation({
    mutationFn: () =>
      unwrap(
        api.DELETE(
          "/universities/{slug}/sections/{id}/assessments/{assessmentId}",
          {
            params: {
              path: { slug, id: sectionId, assessmentId: assessment!.id },
            },
          },
        ),
      ),
    onSuccess: async () => {
      await refresh();
      onSaved(`${assessment!.name} deleted.`);
    },
  });
  const server = useFieldErrors(m.error, copy);
  const fields = { ...server, ...local };
  const hasLocal = Object.keys(local).length > 0;
  useFocusFirstError(formRef, hasLocal ? local : m.error, bannerRef);

  function submit(e: React.FormEvent) {
    e.preventDefault();
    const errs: Record<string, string> = {};
    const max = parseDecimal(maxPoints);
    const w = parseDecimal(weight);
    const n = name.trim();
    if (!n || n.length > 100) errs.name = copy["validation_failed:name"];
    if (max === undefined || max <= 0)
      errs.maxPoints = copy["validation_failed:maxPoints"];
    if (w === undefined || w > 100)
      errs.weight = copy["validation_failed:weight"];
    setLocal(errs);
    if (Object.keys(errs).length > 0 || max === undefined || w === undefined)
      return;
    if (!assessment)
      return m.mutate({
        name: n,
        maxPoints: max,
        weight: w,
        ...(dueOn && { dueOn }),
      });
    const body = {
      ...(n !== assessment.name && { name: n }),
      ...(max !== assessment.maxPoints && { maxPoints: max }),
      ...(w !== assessment.weight && { weight: w }),
      ...(dueOn !== (assessment.dueOn ?? "") && { dueOn: dueOn || null }),
    };
    if (Object.keys(body).length === 0) return onClose();
    m.mutate(body);
  }

  if (confirmDelete && assessment)
    return (
      <Dialog open onClose={onClose} title={`Delete ${assessment.name}?`}>
        <div className="flex flex-col gap-4">
          <FormErrors error={del.error} copy={copy} />
          <p className="text-sm">
            Delete {assessment.name}? Its {scoredCount} scores are deleted too.
            This can&apos;t be undone.
          </p>
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setConfirmDelete(false)}>
              Cancel
            </Button>
            <Button
              variant="danger"
              disabled={del.isPending}
              onClick={() => del.mutate()}
            >
              {del.isPending ? "Deleting…" : "Delete"}
            </Button>
          </div>
        </div>
      </Dialog>
    );

  return (
    <Dialog
      open
      onClose={onClose}
      title={assessment ? `Edit ${assessment.name}` : "New assessment"}
    >
      <form
        ref={formRef}
        noValidate
        onSubmit={submit}
        className="flex flex-col gap-4"
      >
        {Object.keys(fields).length === 0 && (
          <FormErrors ref={bannerRef} error={m.error} copy={copy} />
        )}
        <FormField label="Name" name="name" error={fields.name}>
          <input
            type="text"
            className={inputClass}
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </FormField>
        <FormField
          label="Max points"
          name="maxPoints"
          hint="Points a full score is worth."
          error={fields.maxPoints}
        >
          <input
            type="text"
            inputMode="decimal"
            className={`${inputClass} w-32`}
            value={maxPoints}
            onChange={(e) => setMax(e.target.value)}
          />
        </FormField>
        <FormField
          label="Weight (%)"
          name="weight"
          hint="Share of the section grade. All weights must total 100 to finalize."
          error={fields.weight}
        >
          <input
            type="text"
            inputMode="decimal"
            className={`${inputClass} w-32`}
            value={weight}
            onChange={(e) => setWeight(e.target.value)}
          />
        </FormField>
        <FormField
          label="Due date"
          name="dueOn"
          hint="Optional."
          error={fields.dueOn}
        >
          <input
            type="date"
            className={`${inputClass} w-48`}
            value={dueOn}
            onChange={(e) => setDueOn(e.target.value)}
          />
        </FormField>
        <div className="flex items-center justify-end gap-2">
          {assessment && (
            <Button
              variant="ghost"
              className="mr-auto text-danger hover:bg-danger-soft"
              onClick={() => setConfirmDelete(true)}
            >
              Delete assessment
            </Button>
          )}
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" disabled={m.isPending}>
            {m.isPending
              ? "Saving…"
              : assessment
                ? "Save changes"
                : "Create assessment"}
          </Button>
        </div>
      </form>
    </Dialog>
  );
}
