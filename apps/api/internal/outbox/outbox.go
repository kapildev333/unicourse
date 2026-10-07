// Package outbox stores outgoing mail in the caller's transaction and sends it after commit.
package outbox

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/pg"
)

// Mail is one plain-text message. Urgent mail (sign-in links) is sent before other mail.
type Mail struct {
	To, Subject, Text string
	Urgent            bool
}

// Enqueue stores mails in tx with one statement; nothing is sent unless tx commits. No mails is a no-op.
func Enqueue(ctx context.Context, tx *sql.Tx, mails ...Mail) error {
	if len(mails) == 0 {
		return nil
	}
	to, subject, text, urgent := make([]string, len(mails)), make([]string, len(mails)), make([]string, len(mails)), make([]bool, len(mails))
	for i, m := range mails {
		to[i], subject[i], text[i], urgent[i] = m.To, m.Subject, m.Text, m.Urgent
	}
	_, err := tx.ExecContext(ctx, `insert into outbox_mails (recipient, subject, body, urgent)
select * from unnest($1::text[], $2::text[], $3::text[], $4::bool[])`, to, subject, text, urgent)
	return err
}

// Sender delivers outbox_mails through Mailer. Any number of Senders (processes) may run against one
// database: each row is claimed with FOR UPDATE SKIP LOCKED for the length of its send.
type Sender struct {
	DB          *sql.DB
	Mailer      Mailer
	Log         *slog.Logger
	Poll        time.Duration // fallback when no Wake arrives
	Backoff     time.Duration // delay after the first failure, doubled per attempt, capped at 1h; 0 retries at once (tests)
	MaxAttempts int           // the attempt that reaches this count gives up
	wake        chan struct{}
}

// NewSender sets the defaults: Poll 5s, Backoff 30s, MaxAttempts 8.
func NewSender(db *sql.DB, m Mailer, log *slog.Logger) *Sender {
	return &Sender{DB: db, Mailer: m, Log: log, Poll: 5 * time.Second, Backoff: 30 * time.Second, MaxAttempts: 8, wake: make(chan struct{}, 1)}
}

// Wake nudges Run to drain now. Non-blocking, safe from any goroutine; call it after commit.
func (s *Sender) Wake() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// Run drains, then waits for Wake, the poll tick or the hourly housekeeping tick, until ctx is done.
// It runs housekeeping once at start (processes restarted more often than hourly must still prune).
func (s *Sender) Run(ctx context.Context) {
	s.housekeepLogged(ctx)
	poll, hk := time.NewTicker(s.Poll), time.NewTicker(time.Hour)
	defer poll.Stop()
	defer hk.Stop()
	for {
		if err := s.Drain(ctx); err != nil && ctx.Err() == nil {
			s.Log.Error("mail outbox", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-s.wake:
		case <-poll.C:
		case <-hk.C:
			s.housekeepLogged(ctx)
		}
	}
}

func (s *Sender) housekeepLogged(ctx context.Context) {
	if err := s.housekeep(ctx); err != nil && ctx.Err() == nil {
		s.Log.Error("mail outbox", "err", err)
	}
}

// Drain sends due mail, one transaction per mail, until none is due. It returns the first database
// error, or ctx.Err() when cancelled mid-send (that send is rolled back and is not counted as an attempt).
func (s *Sender) Drain(ctx context.Context) error {
	for {
		claimed, err := s.sendOne(ctx)
		if err != nil || !claimed {
			return err
		}
	}
}

func (s *Sender) sendOne(ctx context.Context) (claimed bool, err error) {
	err = pg.WithPlatformTx(ctx, s.DB, func(tx *sql.Tx) error {
		var id, to, subject, body string
		var attempts int
		err := tx.QueryRowContext(ctx, `select id, recipient, subject, body, attempts from outbox_mails
where failed_at is null and next_attempt_at <= now()
order by urgent desc, next_attempt_at, id
limit 1 for update skip locked`).Scan(&id, &to, &subject, &body, &attempts)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		claimed = true
		// ponytail: one send at a time per process (add goroutines when delivery rate matters).
		// ponytail: at-least-once: a crash between SMTP accept and commit re-sends.
		sendErr := s.Mailer.Send(ctx, to, subject, body)
		if ctx.Err() != nil {
			if sendErr == nil {
				s.Log.Warn("mail sent but rolled back by shutdown, will be sent again", "id", id)
			}
			return ctx.Err() // roll back: row untouched, no attempt counted
		}
		if sendErr == nil {
			_, err = tx.ExecContext(ctx, `delete from outbox_mails where id = $1`, id)
			return err
		}
		var gaveUp bool
		if err := tx.QueryRowContext(ctx, `update outbox_mails set
  attempts = attempts + 1,
  last_error = left($2, 1000),
  next_attempt_at = now() + least(make_interval(secs => $3::float8 * power(2::float8, attempts)), interval '1 hour'),
  failed_at = case when attempts + 1 >= $4 then now() end,
  body = case when attempts + 1 >= $4 then '' else body end
where id = $1
returning failed_at is not null`, id, sendErr.Error(), s.Backoff.Seconds(), s.MaxAttempts).Scan(&gaveUp); err != nil {
			return err
		}
		if gaveUp {
			s.Log.Error("mail gave up", "id", id, "attempts", attempts+1, "err", sendErr)
		} else {
			s.Log.Warn("mail send failed", "id", id, "attempt", attempts+1, "err", sendErr)
		}
		return nil
	})
	return claimed, err
}

func (s *Sender) housekeep(ctx context.Context) error {
	if _, err := s.DB.ExecContext(ctx, `delete from outbox_mails where failed_at < now() - interval '30 days'`); err != nil {
		return err
	}
	var n, age int64
	if err := s.DB.QueryRowContext(ctx, `select count(*), coalesce(extract(epoch from now() - min(created_at)), 0)::bigint
from outbox_mails where failed_at is null and created_at < now() - interval '15 minutes'`).Scan(&n, &age); err != nil {
		return err
	}
	if n > 0 {
		s.Log.Warn("mail outbox backlog", "pending", n, "oldestSeconds", age)
	}
	return nil
}
