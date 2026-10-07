# Postman Collection — Implementation Plan

> **For agentic workers:** executed by `/orchestrate` (orchestrator + `worker` → `reviewer`; the design is already written). One lane. Global constraints, gates and git rules are those of `docs/plans/2026-10-05-platform-core-and-staffing.md`.

**Goal:** a Postman collection and environment, generated from the contract and kept in sync by CI, that signs in as a demo persona and can call any route (Spec 8 §1).

**Architecture:** `apps/web/scripts/postman.mjs` converts `apps/api/openapi.json` with `openapi-to-postmanv2` and normalises the output (design §4); CI regenerates and diffs; a hand-written environment supplies the local defaults.

**Tech Stack:** one new dev dependency (`openapi-to-postmanv2`, pinned) in `apps/web`; Vitest for the unit test; Newman via `npx` for verification only.

**Spec:** `docs/specs/2026-10-07-postman-collection.md`; design `docs/design/P12-L1.md`.

## Global Constraints

- Nothing under `apps/api`; the spec is the single source.
- The generated file is byte-identical across runs; CI's diff is the drift gate.
- Test-driven: `scripts/postman.test.mjs` is written and failing before `postman.mjs`; every normalisation rule in design §4 has an assertion.
- Gates: `npm run lint`, `npm run typecheck`, `npm test -- --run`, `npm run build`, `npm run postman` twice with a clean `git diff --exit-code` after the second, and the Newman real run of spec §5.

## Review Focus

- Determinism (two runs identical), the auth stripping and the cookie scripts, the exit-1 paths for missing routes.
- The README section names match the generated folder and request names.
- The dev dependency is pinned and nothing else was added.

## Phase 12 — Postman collection

### Task 1 (P12-L1, W): generator, collection, environment, CI, README

Spec 8 §3 to §5, §7; design §3 to §8. Scope: `apps/web/scripts/**`, `apps/web/package.json`, `apps/web/package-lock.json`, `docs/postman/**`, `.github/workflows/ci.yml` (the web job only), `README.md` (one new section). Reviewer: Review Focus plus a rerun of the Newman flow.
