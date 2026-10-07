-- Outgoing mail. Written inside the caller's transaction (outbox.Enqueue), sent after commit by
-- outbox.Sender in every API process. Platform table outside row level security, like users,
-- sessions and magic_links: sign-in mail has no university, the sender reads all rows, and no API
-- route reads it. body holds sign-in and invitation URLs in plain text, so a row is deleted once
-- sent and its body is cleared when the sender gives up (failed_at set).
create table outbox_mails (
  id uuid primary key default gen_random_uuid(),
  recipient text not null,
  subject text not null,
  body text not null,
  urgent boolean not null default false,
  attempts integer not null default 0,
  next_attempt_at timestamptz not null default now(),
  last_error text,
  failed_at timestamptz,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
create index on outbox_mails (urgent desc, next_attempt_at, id) where failed_at is null;
create index on outbox_mails (failed_at) where failed_at is not null;
create trigger outbox_mails_set_updated_at before update on outbox_mails for each row execute function set_updated_at();
