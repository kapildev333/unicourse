-- Spec 12 §3: per-attempt time limit for quizzes (D150), late window and penalty for assignments (D151).
-- The effective deadline of an attempt is computed in SQL (quiz/deadline.go), never stored.
alter table assessments
  add column time_limit_minutes integer,
  add column late_until timestamptz,
  add column late_penalty_percent numeric(5,2) not null default 0,
  add constraint assessments_time_limit_check
    check (time_limit_minutes is null or (kind = 'quiz' and time_limit_minutes between 1 and 1440)),
  add constraint assessments_late_until_check
    check (late_until is null or (kind = 'assignment' and late_until > closes_at)),
  add constraint assessments_late_penalty_check
    check (late_penalty_percent between 0 and 100 and (late_penalty_percent = 0 or kind = 'assignment'));

-- Set by any student write at or after closes_at; never cleared (D151).
alter table submissions add column late boolean not null default false;
