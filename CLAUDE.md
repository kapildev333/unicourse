# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this directory is

UniCourse: a multi-tenant platform for universities to host courses, manage classes and sections, and link professors and TAs. Product/startup, not a toy. No product code yet, so no build, lint, or test commands; add them here as soon as the stack exists. Layout: `specs/` (design specs, approved before planning), `fe/` (frontend), `be/` (backend). Specs go in `specs/`, not `docs/superpowers/specs/`.

## Orchestration workflow

The only shared tooling is a plan-driven multi-agent workflow under `.claude/`:

- `/orchestrate <source documents>` (`.claude/commands/orchestrate.md`) makes the main session the orchestrator. It reads `PLAN.md`, or proposes one from the source documents and stops for discussion before launching anything.
- `PLAN.md` at the root is currently an unfilled template (placeholders in `<angle brackets>`): a decisions table, method rules, phases cut into lanes, backlog, and journal.
- The orchestrator is the main session (Fable 5.1). Lanes are sub-agents with disjoint directory scopes, run in this order per piece of work:
  - `designer` (Opus 5.5) writes the design document when a brief needs design judgement; no product code.
  - `worker` (Sonnet 5.5) implements the brief against that design; stops and reports rather than making an open design choice.
  - `reviewer` (Opus 5.5) reviews the landing read-only against brief and design, re-running gates itself.

Shared memory between lanes lives in `.claude/lanes/<LANE-ID>.md` (format in `.claude/lanes/README.md`): each lane writes only its own file, reads the others at start and finish, and addresses requests to other lanes there. The orchestrator reads all of them, relays urgent requests (SendMessage to a running lane, otherwise in the next brief), and folds anything durable into `PLAN.md`.

Rules that span these files:

- Only the orchestrator edits `PLAN.md`. Lanes escalate plan-worthy decisions in their report.
- Lanes work only inside the paths their brief lists; anything needed outside is reported, not edited.
- One directory has one owner at a time, and only one integration suite runs at a time (orchestrator included).
- The orchestrator verifies every landing itself (re-run suites, then exercise the real database/HTTP/browser) and records the evidence in the lane's status cell. A lane's own report is never verification.
- The orchestrator does not write features itself while a lane could.
- Commit locally after each verified landing; never push.
