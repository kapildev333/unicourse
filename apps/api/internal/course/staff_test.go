package course

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/kapildev333/unicourse/apps/api/internal/auth"
)

func (e *env) staffPath(sec string) string { return "/sections/" + sec + "/staff" }

func (e *env) count(q string, args ...any) (n int) {
	e.T.Helper()
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error { return tx.QueryRow(q, args...).Scan(&n) })
	return
}

func (e *env) newSection(termID, number string) (id string) {
	e.T.Helper()
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`insert into sections (university_id, course_id, term_id, number) values ($1,$2,$3,$4) returning id`, e.Fx.UniversityA, e.CourseA, termID, number).Scan(&id)
	})
	return
}

func TestStaffAddActive(t *testing.T) {
	e := newEnv(t)
	e.user("newinst@a.edu", "member")
	e.user("newta@a.edu", "member")
	st, raw := e.call(e.DaIn, "POST", e.staffPath(e.SectionA), map[string]any{"email": "  NewInst@A.edu ", "role": "instructor"})
	var got Staff
	if st != 201 || json.Unmarshal(raw, &got) != nil || got.UserID != e.UserID["newinst@a.edu"] || got.Role != "instructor" || got.Email != "newinst@a.edu" || got.DisplayName == "" {
		t.Fatalf("dept_admin adds instructor: %d %s", st, raw)
	}
	if n, _, after := e.audit("staff.create", e.count0(t, e.UserID["newinst@a.edu"])); n != 1 || !after.Valid {
		t.Errorf("audit staff.create: %d %v", n, after)
	}
	if st, raw := e.call(e.Inst, "POST", e.staffPath(e.SectionA), map[string]any{"email": "newta@a.edu", "role": "ta"}); st != 201 {
		t.Errorf("instructor adds ta: %d %s", st, raw)
	}
	for name, c := range map[string]*http.Cookie{"ta": e.TA, "student": e.Stud, "other": e.Other, "daOut": e.DaOut} {
		if st, raw := e.call(c, "POST", e.staffPath(e.SectionA), map[string]any{"email": "other@a.edu", "role": "ta"}); st != 403 {
			t.Errorf("%s adds: %d %s", name, st, raw)
		}
	}
	// another instructor may not add an instructor
	e.user("inst2@a.edu", "member")
	if st, _ := e.call(e.DaIn, "POST", e.staffPath(e.SectionA), map[string]any{"email": "inst2@a.edu", "role": "instructor"}); st != 201 {
		t.Fatal("setup inst2")
	}
	if st, raw := e.call(e.Inst, "POST", e.staffPath(e.SectionA), map[string]any{"email": "other@a.edu", "role": "instructor"}); st != 403 {
		t.Errorf("instructor adds instructor: %d %s", st, raw)
	}
	if st, _ := e.call(e.DaOut, "GET", e.staffPath(e.SectionA), nil); st != 200 {
		t.Errorf("daOut lists (ruling 6): %d", st)
	}
}

// count0 returns the section_staff row id of userID in S1 (the audit entity id).
func (e *env) count0(t *testing.T, userID string) (id string) {
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select id from section_staff where section_id = $1 and user_id = $2`, e.SectionA, userID).Scan(&id)
	})
	return
}

func TestStaffValidationAndDuplicates(t *testing.T) {
	e := newEnv(t)
	for _, b := range []map[string]any{
		{"email": "nope", "role": "ta"}, {"email": "other@a.edu", "role": "student"}, {"email": "other@a.edu"}, {"role": "ta"},
		{"email": nil, "role": "ta"}, {"email": "other@a.edu", "role": "ta", "x": 1}, {"email": "o\u0000@a.edu", "role": "ta"},
	} {
		if st, raw := e.call(e.DaIn, "POST", e.staffPath(e.SectionA), b); st != 400 {
			t.Errorf("%v: %d %s", b, st, raw)
		}
	}
	// check order: unknown section 404 before body 400
	for _, id := range []string{"nope", "00000000-0000-0000-0000-000000000000"} {
		if st, _ := e.call(e.DaIn, "POST", e.staffPath(id), map[string]any{}); st != 404 {
			t.Errorf("unknown section %s: %d", id, st)
		}
	}
	// mixed-case/whitespace resolves to the existing member and duplicates 409
	if st, raw := e.call(e.DaIn, "POST", e.staffPath(e.SectionA), map[string]any{"email": " TA@A.edu ", "role": "ta"}); st != 409 {
		t.Errorf("duplicate staff: %d %s", st, raw)
	}
	if st, raw := e.call(e.DaIn, "POST", e.staffPath(e.SectionA), map[string]any{"email": "OTHER@a.edu", "role": "ta"}); st != 201 {
		t.Errorf("mixed-case existing member: %d %s", st, raw)
	}
}

func TestStaffInviteFlow(t *testing.T) {
	e := newEnv(t)
	s2 := e.newSection(e.TermA2, "001")
	body := map[string]any{"email": "Newbie@Outside.org", "role": "ta"}
	st, raw := e.call(e.Inst, "POST", e.staffPath(e.SectionA), body)
	var got struct {
		Invited    bool
		Invitation struct{ ID, Email, Role string }
	}
	if st != 202 || json.Unmarshal(raw, &got) != nil || !got.Invited || got.Invitation.Email != "newbie@outside.org" || got.Invitation.Role != "member" {
		t.Fatalf("invite: %d %s", st, raw)
	}
	if e.count(`select count(*) from outbox_mails`) != 1 || e.count(`select count(*) from pending_section_staff`) != 1 {
		t.Errorf("mails %d", e.count(`select count(*) from outbox_mails`))
	}
	_, raw = e.call(e.Admin, "GET", "/sections/"+e.SectionA, nil)
	if p := jmap(t, raw)["pendingStaff"].([]any); len(p) != 1 || p[0].(map[string]any)["email"] != "newbie@outside.org" {
		t.Errorf("pendingStaff: %s", raw)
	}
	if st, _ := e.call(e.Inst, "POST", e.staffPath(e.SectionA), body); st != 409 {
		t.Errorf("duplicate pending: %d", st)
	}
	// same email, second section: reuse the invitation
	if st, raw := e.call(e.DaIn, "POST", e.staffPath(s2), map[string]any{"email": "newbie@outside.org", "role": "instructor"}); st != 202 {
		t.Fatalf("second section: %d %s", st, raw)
	}
	if n := e.count(`select count(*) from invitations where email = 'newbie@outside.org'`); n != 1 || e.count(`select count(*) from outbox_mails`) != 1 {
		t.Errorf("invitations %d mails %d", n, e.count(`select count(*) from outbox_mails`))
	}
	if n := e.count(`select count(*) from pending_section_staff`); n != 2 {
		t.Errorf("pending rows %d", n)
	}
	uid, err := auth.SignIn(context.Background(), e.DB, "newbie@outside.org", "Newbie")
	if err != nil {
		t.Fatal(err)
	}
	if n := e.count(`select count(*) from section_staff where user_id = $1`, uid); n != 2 {
		t.Errorf("section_staff after sign-in: %d", n)
	}
	_, raw = e.call(e.Admin, "GET", "/sections/"+e.SectionA, nil)
	if p := jmap(t, raw)["pendingStaff"].([]any); len(p) != 0 {
		t.Errorf("pendingStaff after accept: %s", raw)
	}
}

func TestStaffIgnoresExpiredInvitation(t *testing.T) {
	e := newEnv(t)
	var oldID string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`insert into invitations (university_id, email, role, token_hash, expires_at)
			values ($1, 'late@outside.org', 'member', '\x01', now() - interval '1 hour') returning id`, e.Fx.UniversityA).Scan(&oldID)
	})
	st, raw := e.call(e.Inst, "POST", e.staffPath(e.SectionA), map[string]any{"email": "late@outside.org", "role": "ta"})
	var got struct{ Invitation struct{ ID string } }
	if st != 202 || json.Unmarshal(raw, &got) != nil || got.Invitation.ID == "" || got.Invitation.ID == oldID {
		t.Fatalf("expired invitation reused: %d %s (old %s)", st, raw, oldID)
	}
	if e.count(`select count(*) from outbox_mails`) != 1 {
		t.Errorf("mails %d", e.count(`select count(*) from outbox_mails`))
	}
	if n := e.count(`select count(*) from pending_section_staff where invitation_id = $1`, got.Invitation.ID); n != 1 {
		t.Errorf("pending rows on new invitation %d", n)
	}
	if n := e.count(`select count(*) from pending_section_staff where invitation_id = $1`, oldID); n != 0 {
		t.Errorf("pending rows on expired invitation %d", n)
	}
}

func TestStaffConcurrentAddsSameEmail(t *testing.T) {
	e := newEnv(t)
	const n = 8
	codes := make([]int, n)
	var wg sync.WaitGroup
	for i := range codes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes[i], _ = e.call(e.Inst, "POST", e.staffPath(e.SectionA), map[string]any{"email": "race@outside.org", "role": "ta"})
		}()
	}
	wg.Wait()
	ok, conflict := 0, 0
	for _, c := range codes {
		switch c {
		case 202:
			ok++
		case 409:
			conflict++
		}
	}
	if ok != 1 || conflict != n-1 {
		t.Errorf("codes %v", codes)
	}
	if c := e.count(`select count(*) from invitations where email = 'race@outside.org'`); c != 1 {
		t.Errorf("invitations %d", c)
	}
	if c := e.count(`select count(*) from pending_section_staff`); c != 1 {
		t.Errorf("pending rows %d", c)
	}
}

func TestStaffDelete(t *testing.T) {
	e := newEnv(t)
	instID, taID := e.UserID["inst@a.edu"], e.UserID["ta@a.edu"]
	path := func(u string) string { return e.staffPath(e.SectionA) + "/" + u }
	if st, _ := e.call(e.Inst, "DELETE", path(instID), nil); st != 403 {
		t.Errorf("instructor removes self: %d", st)
	}
	if st, _ := e.call(e.TA, "DELETE", path(taID), nil); st != 403 {
		t.Errorf("ta removes: %d", st)
	}
	if st, _ := e.call(e.DaOut, "DELETE", path(taID), nil); st != 403 {
		t.Errorf("daOut removes: %d", st)
	}
	for _, u := range []string{"nope", "00000000-0000-0000-0000-000000000000", e.UserID["other@a.edu"]} {
		if st, _ := e.call(e.Inst, "DELETE", path(u), nil); st != 404 {
			t.Errorf("unknown %s: %d", u, st)
		}
	}
	rowID := e.count0(t, taID)
	if st, raw := e.call(e.Inst, "DELETE", path(taID), nil); st != 204 {
		t.Fatalf("instructor removes ta: %d %s", st, raw)
	}
	if n, before, _ := e.audit("staff.delete", rowID); n != 1 || !before.Valid {
		t.Errorf("audit: %d %v", n, before)
	}
	if st, _ := e.call(e.DaIn, "DELETE", path(instID), nil); st != 204 {
		t.Errorf("dept_admin removes instructor: %d", st)
	}
}

func TestStaffListPaginated(t *testing.T) {
	e := newEnv(t)
	for i := 0; i < 3; i++ {
		em := fmt.Sprintf("x%d@a.edu", i)
		e.user(em, "member")
		if st, _ := e.call(e.DaIn, "POST", e.staffPath(e.SectionA), map[string]any{"email": em, "role": "ta"}); st != 201 {
			t.Fatal("setup")
		}
	}
	seen, cursor := map[string]bool{}, ""
	for pages := 0; pages < 10; pages++ {
		st, raw := e.call(e.Stud, "GET", e.staffPath(e.SectionA)+"?limit=2"+cursor, nil)
		var l struct {
			Items      []Staff
			NextCursor string
		}
		if st != 200 || json.Unmarshal(raw, &l) != nil || len(l.Items) > 2 {
			t.Fatalf("list: %d %s", st, raw)
		}
		for _, s := range l.Items {
			seen[s.UserID] = true
		}
		if l.NextCursor == "" {
			break
		}
		cursor = "&cursor=" + l.NextCursor
	}
	if len(seen) != 5 {
		t.Errorf("saw %d staff, want 5", len(seen))
	}
	if st, _ := e.call(e.Other, "GET", e.staffPath(e.SectionA), nil); st != 403 {
		t.Errorf("unrelated member: %d", st)
	}
}

func TestPendingStaffDelete(t *testing.T) {
	e := newEnv(t)
	s2 := e.newSection(e.TermA2, "001")
	mk := func(sec, email, role string) string {
		st, raw := e.call(e.Admin, "POST", e.staffPath(sec), map[string]any{"email": email, "role": role})
		var got struct{ Invitation struct{ ID string } }
		if st != 202 || json.Unmarshal(raw, &got) != nil {
			t.Fatalf("invite: %d %s", st, raw)
		}
		return got.Invitation.ID
	}
	taInv := mk(e.SectionA, "pta@outside.org", "ta")
	instInv := mk(e.SectionA, "pinst@outside.org", "instructor")
	mk(s2, "pta@outside.org", "instructor") // same invitation, other section
	path := func(sec, inv string) string { return "/sections/" + sec + "/pending-staff/" + inv }
	if st, _ := e.call(e.TA, "DELETE", path(e.SectionA, taInv), nil); st != 403 {
		t.Errorf("ta: %d", st)
	}
	if st, _ := e.call(e.DaOut, "DELETE", path(e.SectionA, taInv), nil); st != 403 {
		t.Errorf("daOut: %d", st)
	}
	if st, _ := e.call(e.Inst, "DELETE", path(e.SectionA, instInv), nil); st != 403 {
		t.Errorf("instructor cancels instructor row: %d", st)
	}
	for _, p := range []string{path(e.SectionA, "nope"), path(e.SectionA, "00000000-0000-0000-0000-000000000000"), path(s2, instInv), path("nope", taInv)} {
		if st, _ := e.call(e.Admin, "DELETE", p, nil); st != 404 {
			t.Errorf("unknown %s: %d", p, st)
		}
	}
	_, raw := e.call(e.Admin, "GET", "/sections/"+e.SectionA, nil)
	var sec struct {
		PendingStaff []struct{ InvitationID, Email string }
	}
	if json.Unmarshal(raw, &sec) != nil {
		t.Fatal(string(raw))
	}
	for _, p := range sec.PendingStaff {
		if p.Email == "pta@outside.org" && p.InvitationID != taInv {
			t.Errorf("pendingStaff invitationId %q, want %q", p.InvitationID, taInv)
		}
	}
	if len(sec.PendingStaff) != 2 {
		t.Fatalf("pendingStaff: %s", raw)
	}
	if st, raw := e.call(e.Inst, "DELETE", path(e.SectionA, taInv), nil); st != 204 {
		t.Fatalf("instructor cancels ta row: %d %s", st, raw)
	}
	if st, _ := e.call(e.Inst, "DELETE", path(e.SectionA, taInv), nil); st != 404 {
		t.Errorf("repeat: %d", st)
	}
	if n := e.count(`select count(*) from pending_section_staff where invitation_id = $1`, taInv); n != 1 {
		t.Errorf("other section row should remain: %d", n)
	}
	if n := e.count(`select count(*) from invitations where id = $1`, taInv); n != 1 {
		t.Errorf("invitation must stay: %d", n)
	}
	if n := e.count(`select count(*) from audit_log where action = 'pending_staff.delete' and entity_type = 'section' and entity_id = $1 and before->>'invitationId' = $2 and before->>'email' = 'pta@outside.org' and before->>'role' = 'ta'`, e.SectionA, taInv); n != 1 {
		t.Errorf("audit: %d", n)
	}
	if st, _ := e.call(e.DaIn, "DELETE", path(e.SectionA, instInv), nil); st != 204 {
		t.Errorf("dept_admin cancels instructor row: %d", st)
	}
}

func TestPendingStaffDeleteStale(t *testing.T) {
	e := newEnv(t)
	s2 := e.newSection(e.TermA2, "001")
	mk := func(sec, email string) string {
		st, raw := e.call(e.Admin, "POST", e.staffPath(sec), map[string]any{"email": email, "role": "ta"})
		var got struct{ Invitation struct{ ID string } }
		if st != 202 || json.Unmarshal(raw, &got) != nil {
			t.Fatalf("invite: %d %s", st, raw)
		}
		return got.Invitation.ID
	}
	path := func(sec, inv string) string { return "/sections/" + sec + "/pending-staff/" + inv }
	inv := mk(e.SectionA, "stale@outside.org")
	mk(s2, "stale@outside.org")
	if _, err := auth.SignIn(context.Background(), e.DB, "stale@outside.org", "Stale"); err != nil {
		t.Fatal(err)
	}
	for _, sec := range []string{e.SectionA, s2} {
		if st, _ := e.call(e.Admin, "DELETE", path(sec, inv), nil); st != 404 {
			t.Errorf("accepted invitation, section %s: %d", sec, st)
		}
	}
	exp := mk(e.SectionA, "exp@outside.org")
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`update invitations set expires_at = now() - interval '1 day' where id = $1`, exp)
		return err
	})
	if st, _ := e.call(e.Admin, "DELETE", path(e.SectionA, exp), nil); st != 404 {
		t.Errorf("expired invitation: %d", st)
	}
}
