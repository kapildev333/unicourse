# P6-L4a - courses lists and detail restyle
Status: done
## Doing
Landed on lane/P6-L4-courses; gates green (lint, typecheck, 179 tests, build). e2e not run (brief).
## Found
- Routes needed no change. Row action Delete/Remove use ghost-danger; header actions carry Upload/Plus icons.
- Section detail title band shows instructors; this duplicates names in Staff, so staff-editor.test.tsx (two tests) now scopes `Ida Ins` to the Staff region. That file is P6-L4b's neighbour (test only, not the editor).
- `prettier --check` warns on untouched files too (repo not prettier-clean); not run with --write.
## Requests
## Blockers
