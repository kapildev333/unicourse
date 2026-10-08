-- D133: unicourse_platform runs unicourse-admin create-university and nothing else. It owns nothing,
-- is subject to row level security, and has exactly the grants create-university needs; no grant on
-- schema_migrations and no CREATE on schema public, so it cannot run migrate. When a migration changes
-- what create-university touches, it updates these grants.
do $$
begin
  if not exists (select 1 from pg_roles where rolname = 'unicourse_platform') then
    raise exception 'role unicourse_platform is missing: as a superuser run  create role unicourse_platform login password ''<secret>'' nosuperuser nobypassrls nocreatedb;  then run unicourse-admin migrate again';
  end if;
end $$;
grant usage on schema public to unicourse_platform;
grant select, insert on universities, invitations to unicourse_platform;
grant select (id, email) on users to unicourse_platform;
grant select on memberships to unicourse_platform;
grant insert on grade_scales, audit_log to unicourse_platform;
-- D135: the tenant boundary is a client-set setting, so the platform role is also confined by restrictive
-- policies (ANDed with tenant_isolation) to rows of a university created in the current transaction
-- (now() is the transaction start, and created_at defaults to it). create-university inserts the
-- university and everything else in one transaction; against any existing university these tables
-- return no rows and reject writes.
create policy platform_new_university_only on invitations as restrictive to unicourse_platform
  using (university_id in (select id from universities where created_at = now()))
  with check (university_id in (select id from universities where created_at = now()));
create policy platform_new_university_only on memberships as restrictive to unicourse_platform
  using (university_id in (select id from universities where created_at = now()))
  with check (university_id in (select id from universities where created_at = now()));
create policy platform_new_university_only on grade_scales as restrictive to unicourse_platform
  using (university_id in (select id from universities where created_at = now()))
  with check (university_id in (select id from universities where created_at = now()));
create policy platform_new_university_only on audit_log as restrictive to unicourse_platform
  using (university_id in (select id from universities where created_at = now()))
  with check (university_id in (select id from universities where created_at = now()));
