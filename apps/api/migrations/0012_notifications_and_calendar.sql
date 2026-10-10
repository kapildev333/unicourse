-- Spec 13 (roadmap R2, R3): in-app notifications, the announcement-mail opt-out, calendar feed tokens.
-- Notifications hold no text: the list joins the source, and a deleted source removes its notifications.
alter table memberships add column announcement_mail boolean not null default true;
alter table announcements add constraint announcements_university_id_id_key unique (university_id, id);

create table notifications (
  id uuid primary key default gen_random_uuid(),
  university_id uuid not null references universities(id) on delete cascade,
  user_id uuid not null,
  kind text not null check (kind in ('announcement', 'assessment_created', 'scores_published', 'grades_finalized')),
  section_id uuid not null,
  announcement_id uuid,
  assessment_id uuid,
  read_at timestamptz,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  check ((kind = 'announcement') = (announcement_id is not null)),
  check ((kind in ('assessment_created', 'scores_published')) = (assessment_id is not null)),
  foreign key (university_id, user_id) references memberships (university_id, user_id) on delete cascade,
  foreign key (university_id, section_id) references sections (university_id, id) on delete cascade,
  foreign key (university_id, announcement_id) references announcements (university_id, id) on delete cascade,
  foreign key (university_id, assessment_id) references assessments (university_id, id) on delete cascade
);
create index on notifications (user_id, created_at desc, id desc);
create index on notifications (user_id) where read_at is null;
create index on notifications (section_id);
create index on notifications (announcement_id) where announcement_id is not null;
create index on notifications (assessment_id) where assessment_id is not null;

-- One feed per member per university; only the SHA-256 of the token is stored (like sessions).
create table calendar_feeds (
  id uuid primary key default gen_random_uuid(),
  university_id uuid not null references universities(id) on delete cascade,
  user_id uuid not null,
  token_hash bytea not null unique,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (university_id, user_id),
  foreign key (university_id, user_id) references memberships (university_id, user_id) on delete cascade
);

alter table notifications enable row level security;
alter table notifications force row level security;
create policy tenant_isolation on notifications
  using (university_id = nullif(current_setting('app.university_id', true), '')::uuid)
  with check (university_id = nullif(current_setting('app.university_id', true), '')::uuid);
create trigger notifications_set_updated_at before update on notifications for each row execute function set_updated_at();

alter table calendar_feeds enable row level security;
alter table calendar_feeds force row level security;
create policy tenant_isolation on calendar_feeds
  using (university_id = nullif(current_setting('app.university_id', true), '')::uuid)
  with check (university_id = nullif(current_setting('app.university_id', true), '')::uuid);
create trigger calendar_feeds_set_updated_at before update on calendar_feeds for each row execute function set_updated_at();
