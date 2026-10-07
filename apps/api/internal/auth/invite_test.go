package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/pg"
	"github.com/kapildev333/unicourse/apps/api/internal/testdb"
)

// tcount runs a one-int query in a tenant transaction for uni.
func tcount(t *testing.T, db *sql.DB, uni, q string, args ...any) (n int) {
	t.Helper()
	if err := pg.WithTx(context.Background(), db, uni, func(tx *sql.Tx) error {
		return tx.QueryRow(q, args...).Scan(&n)
	}); err != nil {
		t.Fatal(err)
	}
	return n
}

func texec(t *testing.T, db *sql.DB, uni, q string, args ...any) {
	t.Helper()
	if err := pg.WithTx(context.Background(), db, uni, func(tx *sql.Tx) error {
		_, err := tx.Exec(q, args...)
		return err
	}); err != nil {
		t.Fatal(err)
	}
}

// insertInvitation returns the invitation id.
func insertInvitation(t *testing.T, db *sql.DB, uni, email, role string, deptIDs []string, expiresAt time.Time) (id string) {
	t.Helper()
	_, hash, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if err := pg.WithTx(context.Background(), db, uni, func(tx *sql.Tx) error {
		return tx.QueryRow(`insert into invitations (university_id, email, role, department_ids, token_hash, expires_at)
			values ($1, $2, $3, string_to_array($4, ',')::uuid[], $5, $6) returning id`,
			uni, email, role, strings.Join(deptIDs, ","), hash, expiresAt).Scan(&id)
	}); err != nil {
		t.Fatal(err)
	}
	return id
}

func insertSection(t *testing.T, db *sql.DB, fx testdb.Fixture, code string) (id string) {
	t.Helper()
	if err := pg.WithTx(context.Background(), db, fx.UniversityA, func(tx *sql.Tx) error {
		var course string
		if err := tx.QueryRow(`insert into courses (university_id, department_id, code, title) values ($1, $2, $3, 'T') returning id`,
			fx.UniversityA, fx.DeptA, code).Scan(&course); err != nil {
			return err
		}
		return tx.QueryRow(`insert into sections (university_id, course_id, term_id, number) values ($1, $2, $3, '1') returning id`,
			fx.UniversityA, course, fx.TermA).Scan(&id)
	}); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestSignInAcceptsInvitationOnce(t *testing.T) {
	db := testdb.Open(t)
	fx := testdb.Seed(t, db)
	ctx := context.Background()
	insertInvitation(t, db, fx.UniversityA, "bob@a.edu", "dept_admin", []string{fx.DeptA}, time.Now().Add(time.Hour))

	uid, err := SignIn(ctx, db, "  Bob@A.edu ", "Bob")
	if err != nil {
		t.Fatal(err)
	}
	check := func() {
		t.Helper()
		if n := sqlCount(t, db, `select count(*) from users where email = 'bob@a.edu'`); n != 1 {
			t.Fatalf("users %d", n)
		}
		if n := tcount(t, db, fx.UniversityA, `select count(*) from memberships where user_id = $1 and role = 'dept_admin' and status = 'active'`, uid); n != 1 {
			t.Fatalf("memberships %d", n)
		}
		if n := tcount(t, db, fx.UniversityA, `select count(*) from department_admins where user_id = $1 and department_id = $2`, uid, fx.DeptA); n != 1 {
			t.Fatalf("department_admins %d", n)
		}
		if n := tcount(t, db, fx.UniversityA, `select count(*) from invitations where email = 'bob@a.edu' and accepted_at is not null`); n != 1 {
			t.Fatalf("accepted %d", n)
		}
	}
	check()
	uid2, err := SignIn(ctx, db, "bob@a.edu", "Other")
	if err != nil || uid2 != uid {
		t.Fatalf("second: %v %s", err, uid2)
	}
	check()
	var name string
	if err := pg.WithPlatformTx(ctx, db, func(tx *sql.Tx) error {
		return tx.QueryRow(`select display_name from users where id = $1`, uid).Scan(&name)
	}); err != nil || name != "Bob" {
		t.Fatalf("name %q %v", name, err)
	}
	// An admin removes Bob; signing in again must not bring the membership back.
	texec(t, db, fx.UniversityA, `delete from department_admins where user_id = $1`, uid)
	texec(t, db, fx.UniversityA, `delete from memberships where user_id = $1`, uid)
	if _, err := SignIn(ctx, db, "bob@a.edu", ""); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{`select count(*) from memberships where user_id = $1`, `select count(*) from department_admins where user_id = $1`} {
		if n := tcount(t, db, fx.UniversityA, q, uid); n != 0 {
			t.Fatalf("removed access came back: %s = %d", q, n)
		}
	}
}

func TestSignInActivatesInvitedMembership(t *testing.T) {
	db := testdb.Open(t)
	fx := testdb.Seed(t, db)
	ctx := context.Background()
	uid, err := SignIn(ctx, db, "inv@a.edu", "")
	if err != nil {
		t.Fatal(err)
	}
	texec(t, db, fx.UniversityA, `insert into memberships (university_id, user_id, role, status) values ($1, $2, 'member', 'invited')`, fx.UniversityA, uid)
	insertInvitation(t, db, fx.UniversityA, "inv@a.edu", "member", nil, time.Now().Add(time.Hour))
	if _, err := SignIn(ctx, db, "inv@a.edu", ""); err != nil {
		t.Fatal(err)
	}
	if n := tcount(t, db, fx.UniversityA, `select count(*) from memberships where user_id = $1 and status = 'active'`, uid); n != 1 {
		t.Fatalf("active memberships %d", n)
	}
}

func TestSignInSkipsForeignDepartmentIDs(t *testing.T) {
	db := testdb.Open(t)
	fx := testdb.Seed(t, db)
	exp := time.Now().Add(time.Hour)
	// B's invitation lists A's department: it must not lock the invitee out of A or B.
	insertInvitation(t, db, fx.UniversityB, "multi@other.org", "dept_admin", []string{fx.DeptA}, exp)
	insertInvitation(t, db, fx.UniversityA, "multi@other.org", "dept_admin", []string{fx.DeptA}, exp)
	uid, err := SignIn(context.Background(), db, "multi@other.org", "")
	if err != nil {
		t.Fatal(err)
	}
	for u, want := range map[string]int{fx.UniversityA: 1, fx.UniversityB: 0} {
		if n := tcount(t, db, u, `select count(*) from memberships where user_id = $1 and status = 'active'`, uid); n != 1 {
			t.Fatalf("university %s memberships %d", u, n)
		}
		if n := tcount(t, db, u, `select count(*) from department_admins where user_id = $1`, uid); n != want {
			t.Fatalf("university %s department_admins %d, want %d", u, n, want)
		}
	}
}

func TestSignInAcrossUniversities(t *testing.T) {
	db := testdb.Open(t)
	fx := testdb.Seed(t, db)
	exp := time.Now().Add(time.Hour)
	insertInvitation(t, db, fx.UniversityA, "adj@other.org", "member", nil, exp)
	insertInvitation(t, db, fx.UniversityB, "adj@other.org", "admin", nil, exp)
	uid, err := SignIn(context.Background(), db, "adj@other.org", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range []string{fx.UniversityA, fx.UniversityB} {
		if n := tcount(t, db, u, `select count(*) from memberships where user_id = $1 and status = 'active'`, uid); n != 1 {
			t.Fatalf("university %s memberships %d", u, n)
		}
	}
}

func TestSignInNeverDemotes(t *testing.T) {
	db := testdb.Open(t)
	fx := testdb.Seed(t, db)
	insertInvitation(t, db, fx.UniversityA, "admin@a.edu", "member", nil, time.Now().Add(time.Hour))
	if _, err := SignIn(context.Background(), db, "admin@a.edu", ""); err != nil {
		t.Fatal(err)
	}
	if n := tcount(t, db, fx.UniversityA, `select count(*) from memberships where user_id = $1 and role = 'admin'`, fx.AdminA); n != 1 {
		t.Fatalf("admin demoted")
	}
}

func TestSignInPromotes(t *testing.T) {
	db := testdb.Open(t)
	fx := testdb.Seed(t, db)
	exp := time.Now().Add(time.Hour)
	insertInvitation(t, db, fx.UniversityA, "p@a.edu", "member", nil, exp)
	uid, _ := SignIn(context.Background(), db, "p@a.edu", "")
	insertInvitation(t, db, fx.UniversityA, "p@a.edu", "admin", nil, exp)
	if _, err := SignIn(context.Background(), db, "p@a.edu", ""); err != nil {
		t.Fatal(err)
	}
	if n := tcount(t, db, fx.UniversityA, `select count(*) from memberships where user_id = $1 and role = 'admin'`, uid); n != 1 {
		t.Fatal("not promoted")
	}
}

func TestSignInExpiredInvitation(t *testing.T) {
	db := testdb.Open(t)
	fx := testdb.Seed(t, db)
	ctx := context.Background()
	// a.edu is allowed by domain: signs in, but the expired invitation is not accepted.
	insertInvitation(t, db, fx.UniversityA, "old@a.edu", "member", nil, time.Now().Add(-time.Hour))
	uid, err := SignIn(ctx, db, "old@a.edu", "")
	if err != nil {
		t.Fatal(err)
	}
	if n := tcount(t, db, fx.UniversityA, `select count(*) from memberships where user_id = $1`, uid); n != 0 {
		t.Fatalf("memberships %d", n)
	}
	if n := tcount(t, db, fx.UniversityA, `select count(*) from invitations where email = 'old@a.edu' and accepted_at is null`); n != 1 {
		t.Fatal("expired invitation was accepted")
	}
	// Expired and no domain: not allowed.
	insertInvitation(t, db, fx.UniversityA, "late@other.org", "member", nil, time.Now().Add(-time.Hour))
	if _, err := SignIn(ctx, db, "late@other.org", ""); !errors.Is(err, ErrNotAllowed) {
		t.Fatalf("want ErrNotAllowed, got %v", err)
	}
}

func TestSignInAllowRule(t *testing.T) {
	db := testdb.Open(t)
	fx := testdb.Seed(t, db)
	ctx := context.Background()
	if _, err := SignIn(ctx, db, "nobody@other.org", ""); !errors.Is(err, ErrNotAllowed) {
		t.Fatalf("want ErrNotAllowed, got %v", err)
	}
	if n := sqlCount(t, db, `select count(*) from users where email = 'nobody@other.org'`); n != 0 {
		t.Fatal("user created for disallowed email")
	}
	uid, err := SignIn(ctx, db, "carol@a.edu", "Carol")
	if err != nil {
		t.Fatal(err)
	}
	if n := tcount(t, db, fx.UniversityA, `select count(*) from memberships where user_id = $1`, uid); n != 0 {
		t.Fatal("domain alone must not create a membership")
	}
	if _, err := SignIn(ctx, db, "not an email", ""); err == nil {
		t.Fatal("invalid email accepted")
	}
}

func TestSignInPendingStaffAndEnrollments(t *testing.T) {
	db := testdb.Open(t)
	fx := testdb.Seed(t, db)
	s1, s2 := insertSection(t, db, fx, "CS1"), insertSection(t, db, fx, "CS2")
	inv := insertInvitation(t, db, fx.UniversityA, "dave@a.edu", "member", nil, time.Now().Add(time.Hour))
	texec(t, db, fx.UniversityA, `insert into pending_section_staff (university_id, invitation_id, section_id, role) values ($1, $2, $3, 'ta')`, fx.UniversityA, inv, s1)
	texec(t, db, fx.UniversityA, `insert into pending_enrollments (university_id, invitation_id, section_id) values ($1, $2, $3)`, fx.UniversityA, inv, s2)
	uid, err := SignIn(context.Background(), db, "dave@a.edu", "")
	if err != nil {
		t.Fatal(err)
	}
	if n := tcount(t, db, fx.UniversityA, `select count(*) from section_staff where user_id = $1 and section_id = $2 and role = 'ta'`, uid, s1); n != 1 {
		t.Fatalf("section_staff %d", n)
	}
	if n := tcount(t, db, fx.UniversityA, `select count(*) from enrollments where user_id = $1 and section_id = $2`, uid, s2); n != 1 {
		t.Fatalf("enrollments %d", n)
	}
}

func TestSignInConcurrent(t *testing.T) {
	db := testdb.Open(t)
	fx := testdb.Seed(t, db)
	s := insertSection(t, db, fx, "CS1")
	inv := insertInvitation(t, db, fx.UniversityA, "race@a.edu", "dept_admin", []string{fx.DeptA}, time.Now().Add(time.Hour))
	texec(t, db, fx.UniversityA, `insert into pending_enrollments (university_id, invitation_id, section_id) values ($1, $2, $3)`, fx.UniversityA, inv, s)
	const n = 8
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := SignIn(context.Background(), db, "race@a.edu", "Race")
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if c := sqlCount(t, db, `select count(*) from users where email = 'race@a.edu'`); c != 1 {
		t.Fatalf("users %d", c)
	}
	for q, want := range map[string]int{
		`select count(*) from audit_log where action = 'invitation.accept'`:                                      1,
		`select count(*) from audit_log where action = 'membership.create'`:                                      1,
		`select count(*) from audit_log where action = 'enrollment.create'`:                                      1,
		`select count(*) from memberships m join users u on u.id = m.user_id where u.email = 'race@a.edu'`:       1,
		`select count(*) from department_admins d join users u on u.id = d.user_id where u.email = 'race@a.edu'`: 1,
		`select count(*) from enrollments e join users u on u.id = e.user_id where u.email = 'race@a.edu'`:       1,
		`select count(*) from invitations where email = 'race@a.edu' and accepted_at is not null`:                1,
	} {
		if c := tcount(t, db, fx.UniversityA, q); c != want {
			t.Fatalf("%s = %d, want %d", q, c, want)
		}
	}
}

func TestInvitationURL(t *testing.T) {
	if got := InvitationURL("http://x", "tok", "a+b@a.edu"); got != "http://x/invite/tok?email=a%2Bb%40a.edu" {
		t.Fatal(got)
	}
}

type auditRow struct {
	Action, EntityType, EntityID, Actor string
	Before, After                       map[string]any
}

// auditRows returns uni's audit rows oldest first.
func auditRows(t *testing.T, db *sql.DB, uni string) (out []auditRow) {
	t.Helper()
	if err := pg.WithTx(context.Background(), db, uni, func(tx *sql.Tx) error {
		rs, err := tx.Query(`select action, entity_type, entity_id, coalesce(actor_user_id::text, ''), before, after
			from audit_log order by at, action, entity_id`)
		if err != nil {
			return err
		}
		defer rs.Close()
		for rs.Next() {
			var r auditRow
			var b, a []byte
			if err := rs.Scan(&r.Action, &r.EntityType, &r.EntityID, &r.Actor, &b, &a); err != nil {
				return err
			}
			if b != nil {
				if err := json.Unmarshal(b, &r.Before); err != nil {
					return err
				}
			}
			if a != nil {
				if err := json.Unmarshal(a, &r.After); err != nil {
					return err
				}
			}
			out = append(out, r)
		}
		return rs.Err()
	}); err != nil {
		t.Fatal(err)
	}
	return out
}

func findAudit(t *testing.T, rows []auditRow, action string) auditRow {
	t.Helper()
	var got []auditRow
	for _, r := range rows {
		if r.Action == action {
			got = append(got, r)
		}
	}
	if len(got) != 1 {
		t.Fatalf("%s rows = %d, want 1 (all: %+v)", action, len(got), rows)
	}
	return got[0]
}

func TestSignInAuditAccept(t *testing.T) {
	db := testdb.Open(t)
	fx := testdb.Seed(t, db)
	ctx := context.Background()
	inv := insertInvitation(t, db, fx.UniversityA, "bob@a.edu", "dept_admin", []string{fx.DeptA}, time.Now().Add(time.Hour))
	before := len(auditRows(t, db, fx.UniversityA))
	uid, err := SignIn(ctx, db, "bob@a.edu", "Bob")
	if err != nil {
		t.Fatal(err)
	}
	rows := auditRows(t, db, fx.UniversityA)
	if len(rows) != before+2 {
		t.Fatalf("rows %d, want %d: %+v", len(rows), before+2, rows)
	}
	acc := findAudit(t, rows, "invitation.accept")
	if acc.EntityType != "invitation" || acc.EntityID != inv || acc.Actor != uid || acc.Before != nil ||
		acc.After["email"] != "bob@a.edu" || acc.After["role"] != "dept_admin" ||
		len(acc.After["departmentIds"].([]any)) != 1 || acc.After["departmentIds"].([]any)[0] != fx.DeptA {
		t.Fatalf("accept row %+v", acc)
	}
	var mID string
	if err := pg.WithTx(ctx, db, fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select id from memberships where user_id = $1`, uid).Scan(&mID)
	}); err != nil {
		t.Fatal(err)
	}
	m := findAudit(t, rows, "membership.create")
	if m.EntityType != "membership" || m.EntityID != mID || m.Actor != uid || m.Before != nil ||
		m.After["role"] != "dept_admin" || m.After["status"] != "active" ||
		len(m.After["departmentIds"].([]any)) != 1 || m.After["departmentIds"].([]any)[0] != fx.DeptA {
		t.Fatalf("membership row %+v", m)
	}
	// Idempotent second sign-in and another university's log stay untouched.
	if _, err := SignIn(ctx, db, "bob@a.edu", ""); err != nil {
		t.Fatal(err)
	}
	if n := len(auditRows(t, db, fx.UniversityA)); n != before+2 {
		t.Fatalf("second sign-in wrote rows: %d", n)
	}
}

func TestSignInAuditMembershipUpdate(t *testing.T) {
	db := testdb.Open(t)
	fx := testdb.Seed(t, db)
	ctx := context.Background()
	exp := time.Now().Add(time.Hour)
	insertInvitation(t, db, fx.UniversityA, "p@a.edu", "member", nil, exp)
	uid, _ := SignIn(ctx, db, "p@a.edu", "")
	insertInvitation(t, db, fx.UniversityA, "p@a.edu", "admin", nil, exp)
	if _, err := SignIn(ctx, db, "p@a.edu", ""); err != nil {
		t.Fatal(err)
	}
	rows := auditRows(t, db, fx.UniversityA)
	u := findAudit(t, rows, "membership.update")
	if u.Actor != uid || u.Before["role"] != "member" || u.Before["status"] != "active" ||
		u.After["role"] != "admin" || u.After["status"] != "active" {
		t.Fatalf("update row %+v", u)
	}
	findAudit(t, rows, "membership.create")
	// Equal or lower role on an active membership: accept row only, no membership row.
	insertInvitation(t, db, fx.UniversityA, "p@a.edu", "member", nil, exp)
	n := len(rows)
	if _, err := SignIn(ctx, db, "p@a.edu", ""); err != nil {
		t.Fatal(err)
	}
	rows = auditRows(t, db, fx.UniversityA)
	if len(rows) != n+1 {
		t.Fatalf("rows %d, want %d", len(rows), n+1)
	}
	findAudit(t, rows, "membership.update")
}

func TestSignInAuditPerUniversity(t *testing.T) {
	db := testdb.Open(t)
	fx := testdb.Seed(t, db)
	exp := time.Now().Add(time.Hour)
	ia := insertInvitation(t, db, fx.UniversityA, "adj@other.org", "member", nil, exp)
	ib := insertInvitation(t, db, fx.UniversityB, "adj@other.org", "admin", nil, exp)
	if _, err := SignIn(context.Background(), db, "adj@other.org", ""); err != nil {
		t.Fatal(err)
	}
	for uni, inv := range map[string]string{fx.UniversityA: ia, fx.UniversityB: ib} {
		rows := auditRows(t, db, uni)
		if got := findAudit(t, rows, "invitation.accept"); got.EntityID != inv {
			t.Fatalf("university %s accept entity %s, want %s", uni, got.EntityID, inv)
		}
		findAudit(t, rows, "membership.create")
		if len(rows) != 2 {
			t.Fatalf("university %s rows %+v", uni, rows)
		}
	}
}

func TestSignInAuditStaffAndEnrollments(t *testing.T) {
	db := testdb.Open(t)
	fx := testdb.Seed(t, db)
	s1, s2 := insertSection(t, db, fx, "CS1"), insertSection(t, db, fx, "CS2")
	inv := insertInvitation(t, db, fx.UniversityA, "dave@a.edu", "member", nil, time.Now().Add(time.Hour))
	texec(t, db, fx.UniversityA, `insert into pending_section_staff (university_id, invitation_id, section_id, role) values ($1, $2, $3, 'ta')`, fx.UniversityA, inv, s1)
	texec(t, db, fx.UniversityA, `insert into pending_enrollments (university_id, invitation_id, section_id) values ($1, $2, $3)`, fx.UniversityA, inv, s2)
	uid, err := SignIn(context.Background(), db, "dave@a.edu", "")
	if err != nil {
		t.Fatal(err)
	}
	rows := auditRows(t, db, fx.UniversityA)
	st := findAudit(t, rows, "staff.create")
	var staffID, enrID string
	if err := pg.WithTx(context.Background(), db, fx.UniversityA, func(tx *sql.Tx) error {
		if err := tx.QueryRow(`select id from section_staff where user_id = $1`, uid).Scan(&staffID); err != nil {
			return err
		}
		return tx.QueryRow(`select id from enrollments where user_id = $1`, uid).Scan(&enrID)
	}); err != nil {
		t.Fatal(err)
	}
	if st.EntityType != "staff" || st.EntityID != staffID || st.Actor != uid || len(st.After) != 4 || st.After["sectionId"] != s1 ||
		st.After["userId"] != uid || st.After["email"] != "dave@a.edu" || st.After["role"] != "ta" {
		t.Fatalf("staff row %+v", st)
	}
	en := findAudit(t, rows, "enrollment.create")
	if en.EntityType != "enrollment" || en.EntityID != enrID || en.Actor != uid || len(en.After) != 3 || en.After["sectionId"] != s2 ||
		en.After["userId"] != uid || en.After["email"] != "dave@a.edu" {
		t.Fatalf("enrollment row %+v", en)
	}
	// Already staffed and enrolled: the next invitation's pending rows hit on-conflict and write nothing.
	inv2 := insertInvitation(t, db, fx.UniversityA, "dave@a.edu", "member", nil, time.Now().Add(time.Hour))
	texec(t, db, fx.UniversityA, `insert into pending_section_staff (university_id, invitation_id, section_id, role) values ($1, $2, $3, 'ta')`, fx.UniversityA, inv2, s1)
	texec(t, db, fx.UniversityA, `insert into pending_enrollments (university_id, invitation_id, section_id) values ($1, $2, $3)`, fx.UniversityA, inv2, s2)
	if _, err := SignIn(context.Background(), db, "dave@a.edu", ""); err != nil {
		t.Fatal(err)
	}
	if got := auditRows(t, db, fx.UniversityA); len(got) != len(rows)+1 {
		t.Fatalf("rows %d, want %d: %+v", len(got), len(rows)+1, got)
	}
}

func TestSignInAuditNotAllowedWritesNothing(t *testing.T) {
	db := testdb.Open(t)
	fx := testdb.Seed(t, db)
	if _, err := SignIn(context.Background(), db, "x@nowhere.org", ""); !errors.Is(err, ErrNotAllowed) {
		t.Fatalf("want ErrNotAllowed, got %v", err)
	}
	for _, u := range []string{fx.UniversityA, fx.UniversityB} {
		if n := len(auditRows(t, db, u)); n != 0 {
			t.Fatalf("university %s rows %d", u, n)
		}
	}
}

func strs(v any) (out []string) {
	for _, x := range v.([]any) {
		out = append(out, x.(string))
	}
	return out
}

func insertDept(t *testing.T, db *sql.DB, uni, code string) (id string) {
	t.Helper()
	if err := pg.WithTx(context.Background(), db, uni, func(tx *sql.Tx) error {
		return tx.QueryRow(`insert into departments (university_id, name, code) values ($1, $2, $2) returning id`, uni, code).Scan(&id)
	}); err != nil {
		t.Fatal(err)
	}
	return id
}

func sameSet(got []string, want ...string) bool {
	if len(got) != len(want) {
		return false
	}
	m := map[string]bool{}
	for _, g := range got {
		m[g] = true
	}
	for _, w := range want {
		if !m[w] {
			return false
		}
	}
	return true
}

// A dept_admin invitation for a new department changes the full department set even when the
// membership row itself is untouched; admins accepting one get the same row.
func TestSignInAuditDepartmentAdded(t *testing.T) {
	db := testdb.Open(t)
	fx := testdb.Seed(t, db)
	ctx := context.Background()
	exp := time.Now().Add(time.Hour)
	deptX := insertDept(t, db, fx.UniversityA, "X")
	insertInvitation(t, db, fx.UniversityA, "da@a.edu", "dept_admin", []string{fx.DeptA}, exp)
	uid, err := SignIn(ctx, db, "da@a.edu", "")
	if err != nil {
		t.Fatal(err)
	}
	insertInvitation(t, db, fx.UniversityA, "da@a.edu", "dept_admin", []string{deptX}, exp)
	n := len(auditRows(t, db, fx.UniversityA))
	if _, err := SignIn(ctx, db, "da@a.edu", ""); err != nil {
		t.Fatal(err)
	}
	rows := auditRows(t, db, fx.UniversityA)
	if len(rows) != n+2 {
		t.Fatalf("rows %d, want %d: %+v", len(rows), n+2, rows)
	}
	u := findAudit(t, rows, "membership.update")
	if u.Actor != uid || u.Before["role"] != "dept_admin" || u.Before["status"] != "active" || !sameSet(strs(u.Before["departmentIds"]), fx.DeptA) ||
		u.After["role"] != "dept_admin" || !sameSet(strs(u.After["departmentIds"]), fx.DeptA, deptX) {
		t.Fatalf("update row %+v", u)
	}

	// Active admin accepting a dept_admin invitation: membership untouched, department added.
	insertInvitation(t, db, fx.UniversityA, "admin@a.edu", "dept_admin", []string{deptX}, exp)
	if _, err := SignIn(ctx, db, "admin@a.edu", ""); err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, r := range auditRows(t, db, fx.UniversityA) {
		if r.Action == "membership.update" && r.Actor == fx.AdminA {
			found = true
			if r.Before["role"] != "admin" || len(strs(r.Before["departmentIds"])) != 0 || r.After["role"] != "admin" ||
				!sameSet(strs(r.After["departmentIds"]), deptX) {
				t.Fatalf("admin row %+v", r)
			}
		}
	}
	if !found {
		t.Fatal("no membership.update for the admin")
	}
}

// departmentIds is the full set of departments the user really has: a stale id on the invitation is
// skipped and must not appear.
func TestSignInAuditSkipsStaleDepartment(t *testing.T) {
	db := testdb.Open(t)
	fx := testdb.Seed(t, db)
	stale := "00000000-0000-0000-0000-00000000dead"
	insertInvitation(t, db, fx.UniversityA, "st@a.edu", "dept_admin", []string{fx.DeptA, stale}, time.Now().Add(time.Hour))
	if _, err := SignIn(context.Background(), db, "st@a.edu", ""); err != nil {
		t.Fatal(err)
	}
	m := findAudit(t, auditRows(t, db, fx.UniversityA), "membership.create")
	if !sameSet(strs(m.After["departmentIds"]), fx.DeptA) {
		t.Fatalf("membership row %+v", m)
	}
}

func userName(t *testing.T, db *sql.DB, id string) (n string) {
	t.Helper()
	if err := pg.WithPlatformTx(context.Background(), db, func(tx *sql.Tx) error {
		return tx.QueryRow(`select display_name from users where id = $1`, id).Scan(&n)
	}); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestSignInInvitationName(t *testing.T) {
	ctx := context.Background()
	for _, c := range []struct {
		name, oidc, existing, want string
		expired                    bool
	}{
		{"magic link takes the invitation name", "", "", "Dana CSV", false},
		{"OIDC name wins", "Dana OIDC", "", "Dana OIDC", false},
		{"existing name wins", "", "Set", "Set", false},
		{"expired invitation name is ignored", "", "", "", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			db := testdb.Open(t)
			fx := testdb.Seed(t, db)
			exp := time.Now().Add(time.Hour)
			if c.expired {
				exp = time.Now().Add(-time.Hour)
			}
			if c.existing != "" {
				if err := pg.WithPlatformTx(ctx, db, func(tx *sql.Tx) error {
					_, err := tx.Exec(`insert into users (email, display_name) values ('dana@a.edu', $1)`, c.existing)
					return err
				}); err != nil {
					t.Fatal(err)
				}
			}
			inv := insertInvitation(t, db, fx.UniversityA, "dana@a.edu", "member", nil, exp)
			texec(t, db, fx.UniversityA, `update invitations set display_name = 'Dana CSV' where id = $1`, inv)
			uid, err := SignIn(ctx, db, "dana@a.edu", c.oidc)
			if err != nil {
				t.Fatal(err)
			}
			if got := userName(t, db, uid); got != c.want {
				t.Fatalf("name %q, want %q", got, c.want)
			}
			if !c.expired {
				// invitation.accept + membership.create only: the name fill is not audited
				if n := tcount(t, db, fx.UniversityA, `select count(*) from audit_log`); n != 2 {
					t.Fatalf("audit rows %d, want 2", n)
				}
			}
		})
	}
}

func TestSignInInvitationNameOldestWithName(t *testing.T) {
	db := testdb.Open(t)
	fx := testdb.Seed(t, db)
	insertInvitation(t, db, fx.UniversityA, "dana@a.edu", "member", nil, time.Now().Add(time.Hour))
	b := insertInvitation(t, db, fx.UniversityB, "dana@a.edu", "member", nil, time.Now().Add(time.Hour))
	texec(t, db, fx.UniversityB, `update invitations set display_name = 'Second' where id = $1`, b)
	uid, err := SignIn(context.Background(), db, "dana@a.edu", "")
	if err != nil {
		t.Fatal(err)
	}
	if got := userName(t, db, uid); got != "Second" {
		t.Fatalf("name %q", got)
	}
}
