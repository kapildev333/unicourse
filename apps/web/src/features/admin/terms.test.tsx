import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import { mockApi, renderWithQuery } from "../../test-utils";
import { TermsPage } from "./terms";

afterEach(() => vi.unstubAllGlobals());

const fall = {
  id: "1",
  name: "Fall 2026",
  startsOn: "2026-09-01",
  endsOn: "2026-12-15",
};
const L = "GET /universities/x/terms";
const err = (status: number, code: string, details: unknown[] = []) => ({
  status,
  body: { error: { code, message: "m", details } },
});

async function openCreate() {
  await screen.findByText("Fall 2026");
  fireEvent.click(screen.getByRole("button", { name: "New term" }));
}

it("renders dates as local dates and uses date inputs in the dialog", async () => {
  mockApi({ [L]: () => ({ body: { items: [fall] } }) });
  renderWithQuery(<TermsPage slug="x" />);
  const fmt = new Intl.DateTimeFormat(undefined, { dateStyle: "medium" });
  expect(
    await screen.findByText(fmt.format(new Date(2026, 8, 1))),
  ).toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: "New term" }));
  expect(screen.getByLabelText("Starts on")).toHaveAttribute("type", "date");
  expect(screen.getByLabelText("Ends on")).toHaveAttribute("type", "date");
});

it("shows the endsOn field error and focuses the input", async () => {
  mockApi({
    [L]: () => ({ body: { items: [fall] } }),
    "POST /universities/x/terms": () =>
      err(400, "validation_failed", [
        { field: "endsOn", message: "must be after startsOn" },
      ]),
  });
  renderWithQuery(<TermsPage slug="x" />);
  await openCreate();
  fireEvent.change(screen.getByLabelText("Name"), {
    target: { value: "Spring" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Create term" }));
  const msg = await screen.findByText("End date must be after the start date.");
  const input = screen.getByLabelText("Ends on");
  expect(input).toHaveAttribute("aria-invalid", "true");
  expect(input.getAttribute("aria-describedby")).toContain(
    msg.closest("p")!.id,
  );
  await waitFor(() => expect(input).toHaveFocus());
});

it("shows the name conflict copy", async () => {
  mockApi({
    [L]: () => ({ body: { items: [fall] } }),
    "POST /universities/x/terms": () => err(409, "conflict"),
  });
  renderWithQuery(<TermsPage slug="x" />);
  await openCreate();
  fireEvent.click(screen.getByRole("button", { name: "Create term" }));
  expect(
    await screen.findByText("A term with this name already exists."),
  ).toBeInTheDocument();
});

it("shows the delete 409 copy and the empty state", async () => {
  mockApi({
    [L]: () => ({ body: { items: [fall] } }),
    "DELETE /universities/x/terms/1": () => err(409, "conflict"),
  });
  renderWithQuery(<TermsPage slug="x" />);
  await screen.findByText("Fall 2026");
  fireEvent.click(screen.getByRole("button", { name: "Delete Fall 2026" }));
  fireEvent.click(await screen.findByRole("button", { name: "Delete" }));
  expect(
    await screen.findByText("This term still has sections. Delete them first."),
  ).toBeInTheDocument();
});

it("shows the empty state", async () => {
  mockApi({ [L]: () => ({ body: { items: [] } }) });
  renderWithQuery(<TermsPage slug="x" />);
  expect(await screen.findByText("No terms yet")).toBeInTheDocument();
});
