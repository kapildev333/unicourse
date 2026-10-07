-- D48: the API connects as role unicourse, which owns nothing and so cannot disable row level
-- security, drop policies, truncate (TRUNCATE ignores RLS) or create tables. unicourse_owner owns
-- everything and runs migrations (unicourse-admin migrate). The grants live here, not in a one-off,
-- because dropping and recreating schema public (tests) drops its default privileges too.
-- Fails, on purpose, when run as any role that is not, or is not a member of, unicourse_owner.
revoke create on schema public from public;
grant usage on schema public to unicourse;
grant select, insert, update, delete on all tables in schema public to unicourse;
revoke insert, update, delete on schema_migrations from unicourse;
alter default privileges for role unicourse_owner in schema public
  grant select, insert, update, delete on tables to unicourse;
alter default privileges for role unicourse_owner in schema public
  grant usage, select on sequences to unicourse;
