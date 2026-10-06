-- Spec 2 §3: weekly meeting slots of a section (1 = Monday ... 7 = Sunday, ISO).
-- Times are wall-clock in the university's timezone; nothing converts them.
create table section_meetings (
  id uuid primary key default gen_random_uuid(),
  university_id uuid not null references universities(id) on delete cascade,
  section_id uuid not null,
  weekday smallint not null check (weekday between 1 and 7),
  starts_at time not null,
  ends_at time not null,
  location text not null default '' check (char_length(location) <= 100),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  check (starts_at < ends_at),
  foreign key (university_id, section_id) references sections (university_id, id) on delete cascade
);
create index on section_meetings (section_id);

alter table section_meetings enable row level security;
alter table section_meetings force row level security;
create policy tenant_isolation on section_meetings
  using (university_id = nullif(current_setting('app.university_id', true), '')::uuid)
  with check (university_id = nullif(current_setting('app.university_id', true), '')::uuid);

create trigger section_meetings_set_updated_at before update on section_meetings for each row execute function set_updated_at();
