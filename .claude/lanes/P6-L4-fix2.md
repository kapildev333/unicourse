# P6-L4-fix2 - meeting row one line in section dialog
Status: done
## Doing
Committed.
## Found
Dialog gains `size?: 'md' | 'lg'`; section dialog uses lg (672 px). Meeting row is a 5-column grid with items-start and Remove keeping mt-7.5 (items-end would drop Remove below the inputs when a field error shows). Measured at 1280 px: all five centres y=550.5, light and dark.
## Requests
## Blockers
