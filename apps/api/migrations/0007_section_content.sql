-- Spec 6 §3: section content. File bytes live under UNICOURSE_FILES_DIR/<university_id>/<material_id>
-- (internal/blob), never in the database; the materials row is the source of truth.
alter table sections add column description text not null default '' check (char_length(description) <= 10000);

create table announcements (
  id uuid primary key default gen_random_uuid(),
  university_id uuid not null references universities(id) on delete cascade,
  section_id uuid not null,
  author_user_id uuid not null references users(id),
  title text not null check (char_length(title) between 1 and 200),
  body text not null check (char_length(body) between 1 and 20000),
  emailed boolean not null default false,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  foreign key (university_id, section_id) references sections (university_id, id) on delete cascade
);
create index on announcements (section_id, created_at desc, id desc);

-- position is 1..n per section, kept dense by the API (no unique constraint: moves renumber in one statement).
create table weeks (
  id uuid primary key default gen_random_uuid(),
  university_id uuid not null references universities(id) on delete cascade,
  section_id uuid not null,
  title text not null check (char_length(title) between 1 and 100),
  notes text not null default '' check (char_length(notes) <= 5000),
  position integer not null check (position >= 1),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (university_id, section_id, id),
  foreign key (university_id, section_id) references sections (university_id, id) on delete cascade
);
create index on weeks (section_id, position);

-- The (university_id, section_id, week_id) FK keeps a material's week inside its section and university.
create table materials (
  id uuid primary key default gen_random_uuid(),
  university_id uuid not null references universities(id) on delete cascade,
  section_id uuid not null,
  week_id uuid not null,
  title text not null check (char_length(title) between 1 and 200),
  file_name text not null check (char_length(file_name) between 1 and 255),
  content_type text not null check (content_type <> ''),
  size_bytes bigint not null check (size_bytes between 1 and 104857600),
  sha256 text not null check (sha256 ~ '^[0-9a-f]{64}$'),
  position integer not null check (position >= 1),
  uploaded_by_user_id uuid not null references users(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  foreign key (university_id, section_id) references sections (university_id, id) on delete cascade,
  foreign key (university_id, section_id, week_id) references weeks (university_id, section_id, id) on delete cascade
);
create index on materials (week_id, position);
create index on materials (section_id);

alter table announcements enable row level security;
alter table announcements force row level security;
create policy tenant_isolation on announcements
  using (university_id = nullif(current_setting('app.university_id', true), '')::uuid)
  with check (university_id = nullif(current_setting('app.university_id', true), '')::uuid);

alter table weeks enable row level security;
alter table weeks force row level security;
create policy tenant_isolation on weeks
  using (university_id = nullif(current_setting('app.university_id', true), '')::uuid)
  with check (university_id = nullif(current_setting('app.university_id', true), '')::uuid);

alter table materials enable row level security;
alter table materials force row level security;
create policy tenant_isolation on materials
  using (university_id = nullif(current_setting('app.university_id', true), '')::uuid)
  with check (university_id = nullif(current_setting('app.university_id', true), '')::uuid);

create trigger announcements_set_updated_at before update on announcements for each row execute function set_updated_at();
create trigger weeks_set_updated_at before update on weeks for each row execute function set_updated_at();
create trigger materials_set_updated_at before update on materials for each row execute function set_updated_at();
