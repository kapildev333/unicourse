package quiz

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/pg"
)

// Sweeper submits the open attempts past their deadline (D140). Any number of processes may run one:
// rows are claimed with FOR UPDATE SKIP LOCKED in per-university tenant transactions.
type Sweeper struct {
	DB    *sql.DB
	Log   *slog.Logger
	Poll  time.Duration // 10s
	Batch int           // attempts per transaction, 100
}

// NewSweeper sets the defaults: Poll 10s, Batch 100.
func NewSweeper(db *sql.DB, log *slog.Logger) *Sweeper {
	return &Sweeper{DB: db, Log: log, Poll: 10 * time.Second, Batch: 100}
}

// Run sweeps, then waits for the poll tick, until ctx is done; errors are logged, never fatal.
func (s *Sweeper) Run(ctx context.Context) {
	tick := time.NewTicker(s.Poll)
	defer tick.Stop()
	for {
		if _, err := s.Sweep(ctx); err != nil && ctx.Err() == nil {
			s.Log.Error("quiz sweeper", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}

// Sweep is one pass over every university; it returns how many attempts it submitted.
func (s *Sweeper) Sweep(ctx context.Context) (int, error) {
	var unis []string
	err := pg.WithPlatformTx(ctx, s.DB, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `select id::text from universities order by id`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return err
			}
			unis = append(unis, id)
		}
		return rows.Err()
	})
	if err != nil {
		return 0, err
	}
	total := 0
	var errs []error
	for _, uni := range unis {
		n, err := s.sweepUniversity(ctx, uni)
		total += n
		if err != nil {
			s.Log.Error("quiz sweeper", "universityId", uni, "err", err)
			errs = append(errs, err)
		}
		if n > 0 {
			s.Log.Info("quiz attempts submitted", "universityId", uni, "count", n)
		}
	}
	return total, errors.Join(errs...)
}

// sweepUniversity runs batch transactions until one claims fewer than Batch attempts.
// ponytail: one bad row rolls back its whole batch every pass; a savepoint per attempt if that is ever seen.
func (s *Sweeper) sweepUniversity(ctx context.Context, uni string) (int, error) {
	total := 0
	for {
		claimed, done := 0, 0
		err := pg.WithTx(ctx, s.DB, uni, func(tx *sql.Tx) error {
			type key struct{ assessment, user string }
			var keys []key
			// ponytail: the filter scans every unsubmitted attempt of the university each pass; an index on (submitted_at) where submitted_at is null if that grows
			rows, err := tx.QueryContext(ctx, `select a.assessment_id::text, a.user_id::text from attempts a join assessments q on q.id = a.assessment_id
				where a.submitted_at is null and `+deadlineSQL+` <= now()
				order by `+deadlineSQL+`, a.id limit $1 for update of a skip locked`, s.Batch)
			if err != nil {
				return err
			}
			defer rows.Close()
			for rows.Next() {
				var k key
				if err := rows.Scan(&k.assessment, &k.user); err != nil {
					return err
				}
				keys = append(keys, k)
			}
			if err := rows.Err(); err != nil {
				return err
			}
			rows.Close()
			claimed = len(keys)
			for _, k := range keys {
				switch err := submit(ctx, tx, uni, k.assessment, k.user, true); {
				case errors.Is(err, errAlreadySubmitted):
				case err != nil:
					return err
				default:
					done++
				}
			}
			return nil
		})
		if err != nil {
			return total, err
		}
		total += done
		if claimed < s.Batch {
			return total, nil
		}
	}
}
