package tenant

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"

	"github.com/kapildev333/unicourse/apps/api/internal/auth"
)

// mails returns the logged mail records (to, text).
func (e *env) mails() (out []struct{ To, Text string }) {
	for _, l := range strings.Split(e.Log.String(), "\n") {
		var r struct{ Msg, To, Text string }
		if json.Unmarshal([]byte(l), &r) == nil && r.Msg == "mail" {
			out = append(out, struct{ To, Text string }{r.To, r.Text})
		}
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
	if st, raw := e.do(adm, "POST", pathA+"/invitations", map[string]any{"email": "CAROL@a.edu", "role": "member"}); st != 409 || !strings.Contains(string(raw), "already pending") {
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
	if st, raw := e.do(adm, "POST", pathA+"/invitations", map[string]any{"email": "carol@a.edu", "role": "member"}); st != 409 || !strings.Contains(string(raw), "already a member") {
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
		inv, err := CreateInvitation(context.Background(), tx, NewInvitation{UniversityID: uni, Email: "cli@a.edu", Role: "member"}, e.Deps.Mailer, "http://localhost")
		if err != nil || inv.ID == "" {
			t.Fatalf("%v", err)
		}
		var actors int
		tx.QueryRow(`select count(*) from audit_log where entity_id = $1 and actor_user_id is null`, inv.ID).Scan(&actors)
		if actors != 1 {
			t.Fatalf("audit without actor: %d", actors)
		}
		if _, err := CreateInvitation(context.Background(), tx, NewInvitation{UniversityID: uni, Email: "cli@a.edu", Role: "member"}, e.Deps.Mailer, "http://localhost"); err != ErrInvitationPending {
			t.Fatalf("want ErrInvitationPending, got %v", err)
		}
		return nil
	})
}
