-- Spec 11 §3: quizzes and assignments are assessments with a kind (D138). Their answers live in jsonb
-- on the attempt row; files of submissions are blobs at UNICOURSE_FILES_DIR/<university_id>/<file_id>.
alter table assessments
  add column kind text not null default 'manual' check (kind in ('manual', 'quiz', 'assignment')),
  add column description text not null default '' check (char_length(description) <= 10000),
  add column opens_at timestamptz,
  add column closes_at timestamptz,
  add constraint assessments_window_check check (case when kind = 'manual' then opens_at is null and closes_at is null
    else opens_at is not null and closes_at is not null and opens_at < closes_at end);

-- choices: [{"text": "...", "correct": true}], answers: ["accepted", ...]; shapes validated by the handler (D139).
create table questions (
  id uuid primary key default gen_random_uuid(),
  university_id uuid not null references universities(id) on delete cascade,
  assessment_id uuid not null,
  kind text not null check (kind in ('choice', 'blank')),
  prompt text not null check (char_length(prompt) between 1 and 2000),
  points numeric(7,2) not null check (points > 0 and points <= 1000),
  position integer not null check (position >= 1),
  choices jsonb not null default '[]',
  answers jsonb not null default '[]',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  foreign key (university_id, assessment_id) references assessments (university_id, id) on delete cascade
);
create index on questions (assessment_id, position);

-- One attempt per student (D140). score is set exactly when submitted.
create table attempts (
  id uuid primary key default gen_random_uuid(),
  university_id uuid not null references universities(id) on delete cascade,
  assessment_id uuid not null,
  user_id uuid not null references users(id) on delete cascade,
  started_at timestamptz not null default now(),
  submitted_at timestamptz,
  auto_submitted boolean not null default false,
  answers jsonb not null default '{}',
  grading jsonb not null default '{}',
  score numeric(7,2) check (score >= 0),
  feedback text not null default '' check (char_length(feedback) <= 5000),
  reviewed_by uuid references users(id),
  reviewed_at timestamptz,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  check ((submitted_at is null) = (score is null)),
  unique (university_id, assessment_id, user_id),
  foreign key (university_id, assessment_id) references assessments (university_id, id) on delete cascade
);
create index on attempts (assessment_id, submitted_at);
create index on attempts (user_id);

create table submissions (
  id uuid primary key default gen_random_uuid(),
  university_id uuid not null references universities(id) on delete cascade,
  assessment_id uuid not null,
  user_id uuid not null references users(id) on delete cascade,
  body text not null default '' check (char_length(body) <= 20000),
  submitted_at timestamptz not null default now(),
  score numeric(7,2) check (score >= 0),
  feedback text not null default '' check (char_length(feedback) <= 5000),
  graded_by uuid references users(id),
  graded_at timestamptz,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (university_id, assessment_id, user_id),
  unique (university_id, id),
  foreign key (university_id, assessment_id) references assessments (university_id, id) on delete cascade
);
create index on submissions (user_id);

-- size_bytes has no upper bound here: the limit is UNICOURSE_MAX_UPLOAD_BYTES (blob.Dir.Limit).
create table submission_files (
  id uuid primary key default gen_random_uuid(),
  university_id uuid not null references universities(id) on delete cascade,
  submission_id uuid not null,
  file_name text not null check (char_length(file_name) between 1 and 255),
  content_type text not null check (content_type <> ''),
  size_bytes bigint not null check (size_bytes >= 1),
  sha256 text not null check (sha256 ~ '^[0-9a-f]{64}$'),
  position integer not null check (position >= 1),
  uploaded_at timestamptz not null default now(),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  foreign key (university_id, submission_id) references submissions (university_id, id) on delete cascade
);
create index on submission_files (submission_id, position);

-- materials.size_bytes is bounded by UNICOURSE_MAX_UPLOAD_BYTES (blob.Dir.Limit), which may exceed 100 MiB.
alter table materials drop constraint materials_size_bytes_check, add check (size_bytes >= 1);

alter table questions enable row level security;
alter table questions force row level security;
create policy tenant_isolation on questions
  using (university_id = nullif(current_setting('app.university_id', true), '')::uuid)
  with check (university_id = nullif(current_setting('app.university_id', true), '')::uuid);
create trigger questions_set_updated_at before update on questions for each row execute function set_updated_at();

alter table attempts enable row level security;
alter table attempts force row level security;
create policy tenant_isolation on attempts
  using (university_id = nullif(current_setting('app.university_id', true), '')::uuid)
  with check (university_id = nullif(current_setting('app.university_id', true), '')::uuid);
create trigger attempts_set_updated_at before update on attempts for each row execute function set_updated_at();

alter table submissions enable row level security;
alter table submissions force row level security;
create policy tenant_isolation on submissions
  using (university_id = nullif(current_setting('app.university_id', true), '')::uuid)
  with check (university_id = nullif(current_setting('app.university_id', true), '')::uuid);
create trigger submissions_set_updated_at before update on submissions for each row execute function set_updated_at();

alter table submission_files enable row level security;
alter table submission_files force row level security;
create policy tenant_isolation on submission_files
  using (university_id = nullif(current_setting('app.university_id', true), '')::uuid)
  with check (university_id = nullif(current_setting('app.university_id', true), '')::uuid);
create trigger submission_files_set_updated_at before update on submission_files for each row execute function set_updated_at();

