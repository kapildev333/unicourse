-- B1 (Spec 10): sign-in rate limits shared by every API process. One row per (bucket, client_key),
-- a fixed window that ends at expires_at. Platform table outside row level security, like
-- outbox_mails: no university, and no API route reads it. client_key is a client address or a
-- normalised email, kept at most two windows (auth.Limiter prunes expired rows).
-- unicourse gets DML through the default privileges of 0004.
create table rate_buckets (
  bucket text not null,
  client_key text not null,
  hits integer not null,
  expires_at timestamptz not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  primary key (bucket, client_key)
);
create index on rate_buckets (expires_at);
