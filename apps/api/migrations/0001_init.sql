-- UniCourse initial schema. Tenant-to-tenant foreign keys are composite (university_id, id), so a row
-- can never reference a parent of another university.
-- Every tenant table carries university_id and is protected by
-- forced row level security; the API role must be NOSUPERUSER NOBYPASSRLS.

create table universities (
  id uuid primary key default gen_random_uuid(),
  name text not null,
  slug text not null unique,
  allowed_email_domains text[] not null default '{}',
  timezone text not null,
  locale text not null default 'en-US',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
create index on universities using gin (allowed_email_domains);

create table users (
  id uuid primary key default gen_random_uuid(),
  email text not null unique check (email = lower(email)),
  display_name text not null default '',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table sessions (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  token_hash bytea not null unique,
  expires_at timestamptz not null,
  last_seen_at timestamptz not null default now(),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
create index on sessions (user_id);

create table magic_links (
  id uuid primary key default gen_random_uuid(),
  email text not null check (email = lower(email)),
  token_hash bytea not null unique,
  expires_at timestamptz not null,
  used_at timestamptz,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table memberships (
  id uuid primary key default gen_random_uuid(),
  university_id uuid not null references universities(id) on delete cascade,
  user_id uuid not null references users(id) on delete cascade,
  role text not null check (role in ('admin', 'dept_admin', 'member')),
  status text not null check (status in ('invited', 'active')),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (university_id, user_id)
);
create index on memberships (user_id);
create index on memberships (university_id, created_at, id);

create table departments (
  id uuid primary key default gen_random_uuid(),
  university_id uuid not null references universities(id) on delete cascade,
  name text not null,
  code text not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (university_id, code),
  unique (university_id, id)
);
create index on departments (university_id, created_at, id);

create table department_admins (
  id uuid primary key default gen_random_uuid(),
  university_id uuid not null references universities(id) on delete cascade,
  department_id uuid not null,
  user_id uuid not null references users(id) on delete cascade,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (university_id, department_id, user_id),
  foreign key (university_id, department_id) references departments (university_id, id) on delete restrict
);
create index on department_admins (university_id);
create index on department_admins (user_id);

create table terms (
  id uuid primary key default gen_random_uuid(),
  university_id uuid not null references universities(id) on delete cascade,
  name text not null,
  starts_on date not null,
  ends_on date not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  check (starts_on < ends_on),
  unique (university_id, name),
  unique (university_id, id)
);
create index on terms (university_id, created_at, id);

create table courses (
  id uuid primary key default gen_random_uuid(),
  university_id uuid not null references universities(id) on delete cascade,
  department_id uuid not null,
  code text not null,
  title text not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (university_id, code),
  foreign key (university_id, department_id) references departments (university_id, id) on delete restrict,
  unique (university_id, id)
);
create index on courses (department_id);
create index on courses (university_id, created_at, id);

create table sections (
  id uuid primary key default gen_random_uuid(),
  university_id uuid not null references universities(id) on delete cascade,
  course_id uuid not null,
  term_id uuid not null,
  number text not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (university_id, course_id, term_id, number),
  foreign key (university_id, course_id) references courses (university_id, id) on delete restrict,
  foreign key (university_id, term_id) references terms (university_id, id) on delete restrict,
  unique (university_id, id)
);
create index on sections (term_id);
create index on sections (university_id, created_at, id);

create table section_staff (
  id uuid primary key default gen_random_uuid(),
  university_id uuid not null references universities(id) on delete cascade,
  section_id uuid not null,
  user_id uuid not null references users(id) on delete cascade,
  role text not null check (role in ('instructor', 'ta')),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (university_id, section_id, user_id),
  foreign key (university_id, section_id) references sections (university_id, id) on delete cascade
);
create index on section_staff (user_id);
create index on section_staff (university_id, created_at, id);

create table enrollments (
  id uuid primary key default gen_random_uuid(),
  university_id uuid not null references universities(id) on delete cascade,
  section_id uuid not null,
  user_id uuid not null references users(id) on delete cascade,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (university_id, section_id, user_id),
  foreign key (university_id, section_id) references sections (university_id, id) on delete cascade
);
create index on enrollments (user_id);
create index on enrollments (university_id, created_at, id);

-- department_ids has no foreign key (arrays cannot); deleting a department must strip its id here.
create table invitations (
  id uuid primary key default gen_random_uuid(),
  university_id uuid not null references universities(id) on delete cascade,
  email text not null check (email = lower(email)),
  role text not null check (role in ('admin', 'dept_admin', 'member')),
  department_ids uuid[] not null default '{}',
  token_hash bytea not null unique,
  expires_at timestamptz not null,
  accepted_at timestamptz,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (university_id, id)
);
create index on invitations (email) where accepted_at is null;
create index on invitations (university_id, created_at, id);

create table pending_section_staff (
  id uuid primary key default gen_random_uuid(),
  university_id uuid not null references universities(id) on delete cascade,
  invitation_id uuid not null,
  section_id uuid not null,
  role text not null check (role in ('instructor', 'ta')),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (university_id, invitation_id, section_id),
  foreign key (university_id, invitation_id) references invitations (university_id, id) on delete cascade,
  foreign key (university_id, section_id) references sections (university_id, id) on delete cascade
);
create index on pending_section_staff (section_id);
create index on pending_section_staff (university_id, created_at, id);

create table pending_enrollments (
  id uuid primary key default gen_random_uuid(),
  university_id uuid not null references universities(id) on delete cascade,
  invitation_id uuid not null,
  section_id uuid not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (university_id, invitation_id, section_id),
  foreign key (university_id, invitation_id) references invitations (university_id, id) on delete cascade,
  foreign key (university_id, section_id) references sections (university_id, id) on delete cascade
);
create index on pending_enrollments (section_id);

-- entity_id is not a foreign key: audit rows outlive the entity. actor_user_id is null for CLI writes.
create table audit_log (
  id uuid primary key default gen_random_uuid(),
  university_id uuid not null references universities(id) on delete cascade,
  actor_user_id uuid references users(id) on delete restrict,
  action text not null,
  entity_type text not null,
  entity_id uuid not null,
  before jsonb,
  after jsonb,
  at timestamptz not null default now(),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
create index on audit_log (actor_user_id);
create index on audit_log (university_id, entity_type, entity_id, at);
create index on audit_log (university_id, at, id);

alter table memberships enable row level security;
alter table memberships force row level security;
create policy tenant_isolation on memberships
  using (university_id = nullif(current_setting('app.university_id', true), '')::uuid)
  with check (university_id = nullif(current_setting('app.university_id', true), '')::uuid);

alter table department_admins enable row level security;
alter table department_admins force row level security;
create policy tenant_isolation on department_admins
  using (university_id = nullif(current_setting('app.university_id', true), '')::uuid)
  with check (university_id = nullif(current_setting('app.university_id', true), '')::uuid);

alter table departments enable row level security;
alter table departments force row level security;
create policy tenant_isolation on departments
  using (university_id = nullif(current_setting('app.university_id', true), '')::uuid)
  with check (university_id = nullif(current_setting('app.university_id', true), '')::uuid);

alter table terms enable row level security;
alter table terms force row level security;
create policy tenant_isolation on terms
  using (university_id = nullif(current_setting('app.university_id', true), '')::uuid)
  with check (university_id = nullif(current_setting('app.university_id', true), '')::uuid);

alter table courses enable row level security;
alter table courses force row level security;
create policy tenant_isolation on courses
  using (university_id = nullif(current_setting('app.university_id', true), '')::uuid)
  with check (university_id = nullif(current_setting('app.university_id', true), '')::uuid);

alter table sections enable row level security;
alter table sections force row level security;
create policy tenant_isolation on sections
  using (university_id = nullif(current_setting('app.university_id', true), '')::uuid)
  with check (university_id = nullif(current_setting('app.university_id', true), '')::uuid);

alter table section_staff enable row level security;
alter table section_staff force row level security;
create policy tenant_isolation on section_staff
  using (university_id = nullif(current_setting('app.university_id', true), '')::uuid)
  with check (university_id = nullif(current_setting('app.university_id', true), '')::uuid);

alter table enrollments enable row level security;
alter table enrollments force row level security;
create policy tenant_isolation on enrollments
  using (university_id = nullif(current_setting('app.university_id', true), '')::uuid)
  with check (university_id = nullif(current_setting('app.university_id', true), '')::uuid);

alter table invitations enable row level security;
alter table invitations force row level security;
create policy tenant_isolation on invitations
  using (university_id = nullif(current_setting('app.university_id', true), '')::uuid)
  with check (university_id = nullif(current_setting('app.university_id', true), '')::uuid);

alter table audit_log enable row level security;
alter table audit_log force row level security;
create policy tenant_isolation on audit_log
  using (university_id = nullif(current_setting('app.university_id', true), '')::uuid)
  with check (university_id = nullif(current_setting('app.university_id', true), '')::uuid);

alter table pending_section_staff enable row level security;
alter table pending_section_staff force row level security;
create policy tenant_isolation on pending_section_staff
  using (university_id = nullif(current_setting('app.university_id', true), '')::uuid)
  with check (university_id = nullif(current_setting('app.university_id', true), '')::uuid);

alter table pending_enrollments enable row level security;
alter table pending_enrollments force row level security;
create policy tenant_isolation on pending_enrollments
  using (university_id = nullif(current_setting('app.university_id', true), '')::uuid)
  with check (university_id = nullif(current_setting('app.university_id', true), '')::uuid);

create policy memberships_self on memberships for select
  using (user_id = nullif(current_setting('app.user_id', true), '')::uuid and nullif(current_setting('app.university_id', true), '') is null);
create policy department_admins_self on department_admins for select
  using (user_id = nullif(current_setting('app.user_id', true), '')::uuid and nullif(current_setting('app.university_id', true), '') is null);
create policy invitations_by_email on invitations for select
  using (email = nullif(current_setting('app.email', true), '') and nullif(current_setting('app.university_id', true), '') is null);

-- updated_at is maintained here so handlers never set it by hand.
create function set_updated_at() returns trigger language plpgsql as $$
begin
  new.updated_at = now();
  return new;
end $$;
create trigger universities_set_updated_at before update on universities for each row execute function set_updated_at();
create trigger users_set_updated_at before update on users for each row execute function set_updated_at();
create trigger sessions_set_updated_at before update on sessions for each row execute function set_updated_at();
create trigger magic_links_set_updated_at before update on magic_links for each row execute function set_updated_at();
create trigger memberships_set_updated_at before update on memberships for each row execute function set_updated_at();
create trigger departments_set_updated_at before update on departments for each row execute function set_updated_at();
create trigger department_admins_set_updated_at before update on department_admins for each row execute function set_updated_at();
create trigger terms_set_updated_at before update on terms for each row execute function set_updated_at();
create trigger courses_set_updated_at before update on courses for each row execute function set_updated_at();
create trigger sections_set_updated_at before update on sections for each row execute function set_updated_at();
create trigger section_staff_set_updated_at before update on section_staff for each row execute function set_updated_at();
create trigger enrollments_set_updated_at before update on enrollments for each row execute function set_updated_at();
create trigger invitations_set_updated_at before update on invitations for each row execute function set_updated_at();
create trigger pending_enrollments_set_updated_at before update on pending_enrollments for each row execute function set_updated_at();
create trigger pending_section_staff_set_updated_at before update on pending_section_staff for each row execute function set_updated_at();
create trigger audit_log_set_updated_at before update on audit_log for each row execute function set_updated_at();
