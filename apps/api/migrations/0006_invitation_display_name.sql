-- Spec 5 C1 (D83): the roster CSV's displayName for a person without an account. auth.SignIn copies it
-- into an empty users.display_name; '' means no name, as on users. Never overwritten once set.
alter table invitations add column display_name text not null default '' check (char_length(display_name) <= 100);
