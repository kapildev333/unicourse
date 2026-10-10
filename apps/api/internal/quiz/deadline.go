package quiz

// deadlineSQL is the effective deadline of attempt a of quiz q (D150): started_at plus the limit, never after
// the close. least() ignores NULL, so it is q.closes_at without a limit (and for a missing left-joined attempt).
// Every query using it names attempts "a" and assessments "q".
const deadlineSQL = `least(a.started_at + make_interval(mins => q.time_limit_minutes), q.closes_at)`

// DeadlineSQL is deadlineSQL for other packages (the calendar package's Upcoming query); never a second copy.
const DeadlineSQL = deadlineSQL
