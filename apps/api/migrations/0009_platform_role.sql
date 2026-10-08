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
grant select on users, memberships to unicourse_platform;
grant insert on grade_scales, audit_log to unicourse_platform;
