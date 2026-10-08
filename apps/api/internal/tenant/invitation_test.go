package tenant

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/kapildev333/unicourse/apps/api/internal/auth"
	"github.com/kapildev333/unicourse/apps/api/internal/outbox"
)

// mails returns the queued mail (recipient, body) in creation order.
func (e *env) mails() (out []struct{ To, Text string }) {
	rows, err := e.DB.Query(`select recipient, body from outbox_mails order by created_at, id`)
	if err != nil {
		e.T.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var m struct{ To, Text string }
		if err := rows.Scan(&m.To, &m.Text); err != nil {
			e.T.Fatal(err)
		}
		out = append(out, m)
	}
	return
}

func TestInvitationCreateAndAccept(t *testing.T) {
	e := newServer(t, regTenant)
	uni := e.Fx.UniversityA
	adm := e.adminCookie()

	st, raw := e.do(adm, "POST", pathA+"/invitations", map[string]any{"email": "  Carol@A.EDU ", "role": "dept_admin", "departmentIds": []string{e.Fx.DeptA}})
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var inv Invitation
	if err := dec.Decode(&inv); st != 201 || err != nil || inv.Email != "carol@a.edu" || inv.Role != "dept_admin" || len(inv.DepartmentIDs) != 1 || inv.ExpiresAt.IsZero() || inv.AcceptedAt != nil {
		t.Fatalf("create: %d %s %v", st, raw, err)
	}
	ms := e.mails()
	if len(ms) != 1 || ms[0].To != "carol@a.edu" {
		t.Fatalf("mails: %+v", ms)
	}
	var urlLine string
	for _, l := range strings.Split(ms[0].Text, "\n") {
		if strings.HasPrefix(l, "http://localhost/invite/") {
			urlLine = l
		}
	}
	if urlLine == "" {
		t.Fatalf("no url line: %q", ms[0].Text)
	}
	token := strings.TrimSuffix(strings.TrimPrefix(strings.SplitN(urlLine, "?", 2)[0], "http://localhost/invite/"), "")
	var stored []byte
	e.tx(uni, func(tx *sql.Tx) error {
		return tx.QueryRow(`select token_hash from invitations where id = $1`, inv.ID).Scan(&stored)
	})
	if len(stored) != 32 || bytes.Contains(stored, []byte(token)) {
		t.Fatalf("token hash: %d bytes", len(stored))
	}
	if n, _, a := e.auditRow("invitation.create", inv.ID); n != 1 || !a.Valid {
		t.Fatalf("audit: %d", n)
	}

	// duplicate pending, and mixed-case duplicate
	if st, raw := e.do(adm, "POST", pathA+"/invitations", map[string]any{"email": "CAROL@a.edu", "role": "member"}); st != 409 || !strings.Contains(string(raw), "pending invitation") {
		t.Fatalf("duplicate: %d %s", st, raw)
	}

	// accept end to end
	uid, err := auth.SignIn(context.Background(), e.DB, "carol@a.edu", "")
	if err != nil {
		t.Fatal(err)
	}
	var role, status string
	e.tx(uni, func(tx *sql.Tx) error {
		return tx.QueryRow(`select role, status from memberships where user_id = $1`, uid).Scan(&role, &status)
	})
	if role != "dept_admin" || status != "active" || e.deptAdminRows(uid) != 1 {
		t.Fatalf("accepted: %s %s %d", role, status, e.deptAdminRows(uid))
	}
	// now a member, and the accepted invitation cannot be revoked
	if st, raw := e.do(adm, "POST", pathA+"/invitations", map[string]any{"email": "carol@a.edu", "role": "member"}); st != 409 || !strings.Contains(string(raw), "already has access or a pending invitation") {
		t.Fatalf("member: %d %s", st, raw)
	}
	if st, _ := e.do(adm, "DELETE", pathA+"/invitations/"+inv.ID, nil); st != 409 {
		t.Fatalf("revoke accepted: %d", st)
	}
}

func TestInvitationValidationAndExpiry(t *testing.T) {
	e := newServer(t, regTenant)
	uni := e.Fx.UniversityA
	adm := e.adminCookie()
	for _, c := range []struct {
		body  map[string]any
		field string
		code  int
	}{
		{map[string]any{"email": "nope", "role": "member"}, "email", 400},
		{map[string]any{"email": "x@a.edu", "role": "boss"}, "role", 400},
		{map[string]any{"email": "x@a.edu", "role": "dept_admin"}, "departmentIds", 400},
		{map[string]any{"email": "x@a.edu", "role": "member", "departmentIds": []string{e.Fx.DeptA}}, "departmentIds", 400},
		{map[string]any{"email": "x@a.edu", "role": "dept_admin", "departmentIds": []string{"00000000-0000-4000-8000-000000000001"}}, "", 404},
	} {
		st, raw := e.do(adm, "POST", pathA+"/invitations", c.body)
		if _, d := errBody(raw); st != c.code || (c.field != "" && d[0].Field != c.field) {
			t.Fatalf("%v: %d %s", c.body, st, raw)
		}
	}
	// a department of university B is unknown here
	var bd string
	e.tx(e.Fx.UniversityB, func(tx *sql.Tx) error {
		return tx.QueryRow(`insert into departments (university_id, name, code) values ($1,'Bio','BIO') returning id`, e.Fx.UniversityB).Scan(&bd)
	})
	if st, _ := e.do(adm, "POST", pathA+"/invitations", map[string]any{"email": "x@a.edu", "role": "dept_admin", "departmentIds": []string{bd}}); st != 404 {
		t.Fatalf("foreign dept: %d", st)
	}
	if len(e.mails()) != 0 {
		t.Fatal("mail sent for rejected invitation")
	}
	// an expired pending invitation does not block
	e.tx(uni, func(tx *sql.Tx) error {
		_, err := tx.Exec(`insert into invitations (university_id, email, role, token_hash, expires_at)
			values ($1, 'old@a.edu', 'member', sha256(gen_random_uuid()::text::bytea), now() - interval '1 day')`, uni)
		return err
	})
	if st, raw := e.do(adm, "POST", pathA+"/invitations", map[string]any{"email": "old@a.edu", "role": "member"}); st != 201 {
		t.Fatalf("after expired: %d %s", st, raw)
	}
}

func TestInvitationDeptAdminRules(t *testing.T) {
	e := newServer(t, regTenant)
	uni := e.Fx.UniversityA
	adm := e.adminCookie()
	_, da := e.user(uni, "da@a.edu", "dept_admin", "active", e.Fx.DeptA)
	_, mc := e.user(uni, "m@a.edu", "member", "active")

	if st, _ := e.do(da, "POST", pathA+"/invitations", map[string]any{"email": "n1@a.edu", "role": "member"}); st != 201 {
		t.Fatalf("dept_admin invites member: %d", st)
	}
	for _, role := range []string{"admin", "dept_admin"} {
		if st, _ := e.do(da, "POST", pathA+"/invitations", map[string]any{"email": "n2@a.edu", "role": role, "departmentIds": []string{}}); st != 403 {
			t.Fatalf("dept_admin invites %s: %d", role, st)
		}
	}
	for _, c := range []struct{ method, path string }{{"GET", "/invitations"}, {"POST", "/invitations"}} {
		if st, _ := e.do(mc, c.method, pathA+c.path, map[string]any{"email": "n3@a.edu", "role": "member"}); st != 403 {
			t.Fatalf("member %s: %d", c.method, st)
		}
	}
	st, raw := e.do(adm, "POST", pathA+"/invitations", map[string]any{"email": "boss@a.edu", "role": "admin"})
	adminInv := idOf(t, raw)
	if st != 201 {
		t.Fatalf("admin invites admin: %d", st)
	}
	var l struct{ Items []Invitation }
	if st, raw = e.do(da, "GET", pathA+"/invitations", nil); st != 200 || json.Unmarshal(raw, &l) != nil || len(l.Items) != 1 || l.Items[0].Role != "member" {
		t.Fatalf("dept_admin list: %d %s", st, raw)
	}
	if st, raw = e.do(adm, "GET", pathA+"/invitations", nil); st != 200 || json.Unmarshal(raw, &l) != nil || len(l.Items) != 2 {
		t.Fatalf("admin list: %d %s", st, raw)
	}
	if st, _ := e.do(da, "DELETE", pathA+"/invitations/"+adminInv, nil); st != 403 {
		t.Fatalf("dept_admin revokes admin invitation: %d", st)
	}
	memInv := l.Items[0].ID
	if l.Items[0].Role != "member" {
		memInv = l.Items[1].ID
	}
	if st, _ := e.do(da, "DELETE", pathA+"/invitations/"+memInv, nil); st != 204 {
		t.Fatalf("revoke member invitation: %d", st)
	}
	if n, b, _ := e.auditRow("invitation.delete", memInv); n != 1 || !b.Valid {
		t.Fatalf("revoke audit: %d", n)
	}
	if st, _ := e.do(adm, "DELETE", pathA+"/invitations/"+memInv, nil); st != 404 {
		t.Fatalf("again: %d", st)
	}
}

func TestCreateInvitationCLI(t *testing.T) {
	e := newServer(t, regTenant)
	uni := e.Fx.UniversityA
	e.tx(uni, func(tx *sql.Tx) error {
		inv, err := CreateInvitation(context.Background(), tx, NewInvitation{UniversityID: uni, Email: "cli@a.edu", Role: "member"}, outbox.Enqueue, "http://localhost")
		if err != nil || inv.ID == "" {
			t.Fatalf("%v", err)
		}
		var actors int
		tx.QueryRow(`select count(*) from audit_log where entity_id = $1 and actor_user_id is null`, inv.ID).Scan(&actors)
		if actors != 1 {
			t.Fatalf("audit without actor: %d", actors)
		}
		if _, err := CreateInvitation(context.Background(), tx, NewInvitation{UniversityID: uni, Email: "cli@a.edu", Role: "member"}, outbox.Enqueue, "http://localhost"); err != ErrInvitationPending {
			t.Fatalf("want ErrInvitationPending, got %v", err)
		}
		return nil
	})
}

func TestCreateInvitationsBatch(t *testing.T) {
	e := newServer(t, regTenant)
	uni := e.Fx.UniversityA
	e.tx(uni, func(tx *sql.Tx) error {
		inv, err := CreateInvitations(context.Background(), tx, uni, e.Fx.AdminA, []string{"b1@a.edu", "b2@a.edu", "b3@a.edu"}, "member", "http://localhost")
		if err != nil || len(inv) != 3 {
			t.Fatalf("%v %v", inv, err)
		}
		var n, withActor int
		tx.QueryRow(`select count(*), count(actor_user_id) from audit_log where action = 'invitation.create' and entity_id = any($1::uuid[])`,
			[]string{inv[0].ID, inv[1].ID, inv[2].ID}).Scan(&n, &withActor)
		if n != 3 || withActor != 3 {
			t.Fatalf("audit rows %d, with actor %d", n, withActor)
		}
		if out, err := CreateInvitations(context.Background(), tx, uni, "", nil, "member", ""); err != nil || len(out) != 0 {
			t.Fatalf("empty: %v %v", out, err)
		}
		return nil
	})
	if ms := e.mails(); len(ms) != 3 || !strings.Contains(ms[0].Text, "http://localhost/invite/") {
		t.Fatalf("mails: %+v", ms)
	}
}

func TestCreateInvitationQueuesMailInTx(t *testing.T) {
	e := newServer(t, regTenant)
	uni := e.Fx.UniversityA
	e.tx(uni, func(tx *sql.Tx) error {
		_, err := CreateInvitation(context.Background(), tx, NewInvitation{UniversityID: uni, Email: "q@a.edu", Role: "member"}, outbox.Enqueue, "http://localhost")
		if err != nil {
			t.Fatal(err)
		}
		return nil
	})
	if ms := e.mails(); len(ms) != 1 || ms[0].To != "q@a.edu" || !strings.Contains(ms[0].Text, "/invite/") {
		t.Fatalf("mails: %+v", ms)
	}
	boom := errors.New("boom")
	e.tx(uni, func(tx *sql.Tx) error {
		_, err := CreateInvitation(context.Background(), tx, NewInvitation{UniversityID: uni, Email: "r@a.edu", Role: "member"},
			func(context.Context, *sql.Tx, ...outbox.Mail) error { return boom }, "http://localhost")
		if !errors.Is(err, boom) {
			t.Fatalf("want boom, got %v", err)
		}
		return nil
	})
}
