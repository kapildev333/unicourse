# P18-L6bf - fix round for 6b (take-quiz page)
Status: done
## Doing
M1, M2, L1, L2, L4, B1 fixed on lane/P18-L6-quiz-web.
## Found
- M1 test needs the zero crossing and the submit failure in separate act() calls, else effects flush after the failure and the bug hides.
- Results cards are articles, no fieldset; B1 touched only the take form (card moved to the li, fieldset borderless).
- useAutosave now sets saved after a successful save even when stopped (so unsaved() is right at zero).
## Requests
none
## Blockers
none
