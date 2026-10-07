package outbox

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/pg"
	"github.com/kapildev333/unicourse/apps/api/internal/testdb"
)

type sent struct{ to, subject, text string }

// rec records sends; fail decides the error per call (1-based), block waits for ctx.
type rec struct {
	mu    sync.Mutex
	got   []sent
	fail  func(call int) error
	block chan struct{} // closed-on-entry signal when non-nil; Send then waits for ctx
	delay time.Duration
}

func (r *rec) Send(ctx context.Context, to, subject, text string) error {
	r.mu.Lock()
	r.got = append(r.got, sent{to, subject, text})
	n := len(r.got)
	r.mu.Unlock()
	if r.block != nil {
		select {
		case r.block <- struct{}{}:
		default:
		}
		<-ctx.Done()
		return ctx.Err()
	}
	if r.delay > 0 {
		time.Sleep(r.delay)
	}
	if r.fail != nil {
		return r.fail(n)
	}
	return nil
}

func (r *rec) calls() []sent {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]sent(nil), r.got...)
}

func setup(t *testing.T, m Mailer) (*Sender, *sql.DB, *bytes.Buffer) {
	t.Helper()
	db := testdb.Open(t)
	var buf bytes.Buffer
	s := NewSender(db, m, slog.New(slog.NewTextHandler(&lockedWriter{w: &buf}, nil)))
	return s, db, &buf
}

type lockedWriter struct {
	mu sync.Mutex
	w  *bytes.Buffer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

func enqueue(t *testing.T, db *sql.DB, mails ...Mail) {
	t.Helper()
	err := pg.WithPlatformTx(context.Background(), db, func(tx *sql.Tx) error { return Enqueue(context.Background(), tx, mails...) })
	if err != nil {
		t.Fatal(err)
	}
}

func count(t *testing.T, db *sql.DB, q string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(q).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestEnqueueRolledBackSendsNothing(t *testing.T) {
	r := &rec{}
	s, db, _ := setup(t, r)
	errRollback := errors.New("rollback")
	err := pg.WithPlatformTx(context.Background(), db, func(tx *sql.Tx) error {
		if err := Enqueue(context.Background(), tx, Mail{To: "a@a.edu", Subject: "s", Text: "t"}); err != nil {
			return err
		}
		return errRollback
	})
	if !errors.Is(err, errRollback) {
		t.Fatal(err)
	}
	if err := s.Drain(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(r.calls()) != 0 || count(t, db, `select count(*) from outbox_mails`) != 0 {
		t.Fatal("rolled back mail was stored or sent")
	}
}

func TestEnqueueSendsOnceAndDeletes(t *testing.T) {
	r := &rec{}
	s, db, _ := setup(t, r)
	want := []sent{{"a@a.edu", "s1", "t1"}, {"b@a.edu", "s2", "t2"}, {"c@a.edu", "s3", "t3"}}
	enqueue(t, db, Mail{To: "a@a.edu", Subject: "s1", Text: "t1"}, Mail{To: "b@a.edu", Subject: "s2", Text: "t2"}, Mail{To: "c@a.edu", Subject: "s3", Text: "t3"})
	enqueue(t, db) // no mails is a no-op
	for range 2 {
		if err := s.Drain(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	got := r.calls()
	if len(got) != 3 {
		t.Fatalf("%d sends, want 3", len(got))
	}
	for _, w := range want {
		found := false
		for _, g := range got {
			found = found || g == w
		}
		if !found {
			t.Errorf("missing send %+v", w)
		}
	}
	if n := count(t, db, `select count(*) from outbox_mails`); n != 0 {
		t.Fatalf("%d rows left", n)
	}
}

func TestFailingMailerRetriesThenGivesUp(t *testing.T) {
	r := &rec{fail: func(int) error { return errors.New("smtp down") }}
	s, db, buf := setup(t, r)
	s.MaxAttempts, s.Backoff = 3, 0
	enqueue(t, db, Mail{To: "a@a.edu", Subject: "s", Text: "secret-url"})
	if err := s.Drain(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(r.calls()) != 3 {
		t.Fatalf("%d calls, want 3", len(r.calls()))
	}
	var attempts int
	var failed bool
	var body, lastErr string
	if err := db.QueryRow(`select attempts, failed_at is not null, body, last_error from outbox_mails`).Scan(&attempts, &failed, &body, &lastErr); err != nil {
		t.Fatal(err)
	}
	if attempts != 3 || !failed || body != "" || !strings.Contains(lastErr, "smtp down") {
		t.Fatalf("row %d %v %q %q", attempts, failed, body, lastErr)
	}
	if !strings.Contains(buf.String(), "mail gave up") || strings.Contains(buf.String(), "secret-url") {
		t.Fatalf("log %q", buf.String())
	}
	if err := s.Drain(context.Background()); err != nil || len(r.calls()) != 3 {
		t.Fatal("given-up row was retried", err)
	}
}

func TestBackoffDelaysRetry(t *testing.T) {
	r := &rec{fail: func(int) error { return errors.New("smtp down") }}
	s, db, _ := setup(t, r)
	enqueue(t, db, Mail{To: "a@a.edu", Subject: "s", Text: "t"})
	if err := s.Drain(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(r.calls()) != 1 {
		t.Fatalf("%d calls", len(r.calls()))
	}
	if n := count(t, db, `select count(*) from outbox_mails where next_attempt_at > now() + interval '20 seconds' and failed_at is null`); n != 1 {
		t.Fatal("retry not delayed")
	}
}

func TestTransientFailureThenSuccess(t *testing.T) {
	r := &rec{fail: func(n int) error {
		if n == 1 {
			return errors.New("blip")
		}
		return nil
	}}
	s, db, _ := setup(t, r)
	s.Backoff = 0
	enqueue(t, db, Mail{To: "a@a.edu", Subject: "s", Text: "t"})
	if err := s.Drain(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(r.calls()) != 2 || count(t, db, `select count(*) from outbox_mails`) != 0 {
		t.Fatalf("%d calls", len(r.calls()))
	}
}

func TestTwoSendersNeverDoubleSend(t *testing.T) {
	r := &rec{delay: time.Millisecond}
	s1, db, _ := setup(t, r)
	s2 := NewSender(db, r, s1.Log)
	mails := make([]Mail, 200)
	for i := range mails {
		mails[i] = Mail{To: fmt.Sprintf("u%d@a.edu", i), Subject: "s", Text: "t"}
	}
	enqueue(t, db, mails...)
	var wg sync.WaitGroup
	for _, s := range []*Sender{s1, s2} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.Drain(context.Background()); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	seen := map[string]int{}
	for _, c := range r.calls() {
		seen[c.to]++
	}
	if len(seen) != 200 || len(r.calls()) != 200 {
		t.Fatalf("%d distinct, %d sends, want 200", len(seen), len(r.calls()))
	}
}

func TestLockedRowIsSkipped(t *testing.T) {
	r := &rec{}
	s, db, _ := setup(t, r)
	enqueue(t, db, Mail{To: "a@a.edu", Subject: "s", Text: "t"})
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	var id string
	if err := tx.QueryRow(`select id from outbox_mails for update`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if err := s.Drain(context.Background()); err != nil || len(r.calls()) != 0 {
		t.Fatalf("locked row sent: %v", err)
	}
	tx.Rollback()
	if err := s.Drain(context.Background()); err != nil || len(r.calls()) != 1 {
		t.Fatalf("after unlock: %d sends, %v", len(r.calls()), err)
	}
}

func TestUrgentFirst(t *testing.T) {
	r := &rec{}
	s, db, _ := setup(t, r)
	enqueue(t, db, Mail{To: "slow@a.edu", Subject: "s", Text: "t"})
	enqueue(t, db, Mail{To: "fast@a.edu", Subject: "s", Text: "t", Urgent: true})
	if err := s.Drain(context.Background()); err != nil {
		t.Fatal(err)
	}
	if c := r.calls(); len(c) != 2 || c[0].to != "fast@a.edu" {
		t.Fatalf("order %+v", c)
	}
}

func TestCancelledSendIsNotAnAttempt(t *testing.T) {
	r := &rec{block: make(chan struct{}, 1)}
	s, db, _ := setup(t, r)
	enqueue(t, db, Mail{To: "a@a.edu", Subject: "s", Text: "t"})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Drain(ctx) }()
	<-r.block
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Drain returned %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Drain did not return")
	}
	if n := count(t, db, `select count(*) from outbox_mails where attempts = 0 and failed_at is null`); n != 1 {
		t.Fatal("cancelled send changed the row")
	}
}

func TestRunWakesAndStops(t *testing.T) {
	r := &rec{}
	s, db, _ := setup(t, r)
	s.Poll = time.Hour
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { s.Run(ctx); close(done) }()
	time.Sleep(100 * time.Millisecond) // let the first drain finish so only Wake can trigger the next
	enqueue(t, db, Mail{To: "a@a.edu", Subject: "s", Text: "t"})
	s.Wake()
	deadline := time.Now().Add(2 * time.Second)
	for len(r.calls()) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if len(r.calls()) != 1 {
		t.Fatal("Wake did not trigger a send")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not stop")
	}
}

func TestHousekeep(t *testing.T) {
	s, db, buf := setup(t, &rec{})
	for _, q := range []string{
		`insert into outbox_mails (recipient, subject, body, failed_at) values ('old@a.edu', 's', '', now() - interval '31 days')`,
		`insert into outbox_mails (recipient, subject, body, failed_at) values ('recent@a.edu', 's', '', now() - interval '1 day')`,
		`insert into outbox_mails (recipient, subject, body, created_at) values ('stuck@a.edu', 's', 't', now() - interval '20 minutes')`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.housekeep(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := count(t, db, `select count(*) from outbox_mails`); n != 2 {
		t.Fatalf("%d rows, want 2", n)
	}
	if n := count(t, db, `select count(*) from outbox_mails where recipient = 'old@a.edu'`); n != 0 {
		t.Fatal("old failed row kept")
	}
	if l := buf.String(); !strings.Contains(l, "mail outbox backlog") || !strings.Contains(l, "pending=1") {
		t.Fatalf("log %q", l)
	}
}
