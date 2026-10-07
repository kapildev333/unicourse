-- Spec 4 §3: grading. Percent and points are numeric; the API never converts them through floats.
alter table courses add column credits numeric(3,1) not null default 3.0 check (credits between 0 and 20);

-- One row per letter; the letter of a percent is the row with the greatest min_percent <= percent.
create table grade_scales (
  id uuid primary key default gen_random_uuid(),
  university_id uuid not null references universities(id) on delete cascade,
  letter text not null check (char_length(letter) between 1 and 10),
  min_percent numeric(5,2) not null check (min_percent between 0 and 100),
  points numeric(3,2) not null check (points >= 0),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (university_id, letter),
  unique (university_id, min_percent)
);

-- The US plus/minus 4.0 preset (spec §1). Called by this migration and by unicourse-admin create-university.
create function seed_us_grade_scale(u uuid) returns void language sql as $$
  insert into grade_scales (university_id, letter, min_percent, points)
  select u, v.letter, v.min_percent, v.points from (values
    ('A', 93.00, 4.00), ('A-', 90.00, 3.70), ('B+', 87.00, 3.30), ('B', 83.00, 3.00), ('B-', 80.00, 2.70),
    ('C+', 77.00, 2.30), ('C', 73.00, 2.00), ('C-', 70.00, 1.70), ('D+', 67.00, 1.30), ('D', 63.00, 1.00),
    ('D-', 60.00, 0.70), ('F', 0.00, 0.00)) as v(letter, min_percent, points)
$$;

-- Existing universities get the preset now, before row level security is switched on for the table
-- (the owner is subject to forced RLS and has no app.university_id here).
select seed_us_grade_scale(id) from universities;

create table assessments (
  id uuid primary key default gen_random_uuid(),
  university_id uuid not null references universities(id) on delete cascade,
  section_id uuid not null,
  name text not null check (char_length(name) between 1 and 100),
  max_points numeric(7,2) not null check (max_points > 0 and max_points <= 10000),
  weight numeric(5,2) not null check (weight between 0 and 100),
  due_on date,
  published boolean not null default false,
  position integer not null default 0 check (position >= 0),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (university_id, section_id, name),
  unique (university_id, id),
  foreign key (university_id, section_id) references sections (university_id, id) on delete cascade
);

-- user_id must be enrolled in the assessment's section: checked by the handler (spec §3), not here,
-- so removing an enrollment keeps the scores (hidden from the gradebook until re-enrolled).
create table scores (
  id uuid primary key default gen_random_uuid(),
  university_id uuid not null references universities(id) on delete cascade,
  assessment_id uuid not null,
  user_id uuid not null references users(id) on delete cascade,
  points numeric(7,2) not null check (points >= 0),
  graded_by uuid not null references users(id),
  graded_at timestamptz not null default now(),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (university_id, assessment_id, user_id),
  foreign key (university_id, assessment_id) references assessments (university_id, id) on delete cascade
);
create index on scores (user_id);

-- A snapshot: letter and points are copied from the scale at finalization; scale edits never touch it.
create table final_grades (
  id uuid primary key default gen_random_uuid(),
  university_id uuid not null references universities(id) on delete cascade,
  section_id uuid not null,
  user_id uuid not null references users(id) on delete cascade,
  status text not null check (status in ('graded', 'incomplete', 'withdrawn')),
  percent numeric(5,2) check (percent >= 0),
  letter text check (char_length(letter) between 1 and 10),
  points numeric(3,2) check (points >= 0),
  credits numeric(3,1) not null check (credits between 0 and 20), -- the course's credits at finalization; GPA reads this
  override_reason text check (char_length(override_reason) between 1 and 500),
  finalized_by uuid not null references users(id),
  finalized_at timestamptz not null default now(),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  check (case when status = 'graded' then percent is not null and letter is not null and points is not null
              else percent is null and letter is null and points is null end),
  unique (university_id, section_id, user_id),
  -- restrict: deleting a section that holds final grades is a 409 (transcript rows are not erased by accident)
  foreign key (university_id, section_id) references sections (university_id, id) on delete restrict
);
create index on final_grades (user_id);

alter table grade_scales enable row level security;
alter table grade_scales force row level security;
create policy tenant_isolation on grade_scales
  using (university_id = nullif(current_setting('app.university_id', true), '')::uuid)
  with check (university_id = nullif(current_setting('app.university_id', true), '')::uuid);

alter table assessments enable row level security;
alter table assessments force row level security;
create policy tenant_isolation on assessments
  using (university_id = nullif(current_setting('app.university_id', true), '')::uuid)
  with check (university_id = nullif(current_setting('app.university_id', true), '')::uuid);

alter table scores enable row level security;
alter table scores force row level security;
create policy tenant_isolation on scores
  using (university_id = nullif(current_setting('app.university_id', true), '')::uuid)
  with check (university_id = nullif(current_setting('app.university_id', true), '')::uuid);

alter table final_grades enable row level security;
alter table final_grades force row level security;
create policy tenant_isolation on final_grades
  using (university_id = nullif(current_setting('app.university_id', true), '')::uuid)
  with check (university_id = nullif(current_setting('app.university_id', true), '')::uuid);

create trigger grade_scales_set_updated_at before update on grade_scales for each row execute function set_updated_at();
create trigger assessments_set_updated_at before update on assessments for each row execute function set_updated_at();
create trigger scores_set_updated_at before update on scores for each row execute function set_updated_at();
create trigger final_grades_set_updated_at before update on final_grades for each row execute function set_updated_at();
