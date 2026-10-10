package tenant

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/audit"
)

func TestAuditLogRoute(t *testing.T) {
	e := newServer(t, regTenant)
	uni := e.Fx.UniversityA
	adm := e.adminCookie()
	_, da := e.user(uni, "da@a.edu", "dept_admin", "active", e.Fx.DeptA)

	st, raw := e.do(adm, "POST", pathA+"/departments", map[string]any{"name": "Physics", "code": "PHY"})
	id := idOf(t, raw)
	e.do(adm, "PATCH", pathA+"/departments/"+id, map[string]any{"name": "Physics 2"})
	e.do(adm, "DELETE", pathA+"/departments/"+id, nil)
	if st != 201 {
		t.Fatalf("create: %d", st)
	}

	get := func(q string) (int, []audit.Row, string) {
		st, raw := e.do(adm, "GET", pathA+"/audit-log"+q, nil)
		var l struct {
			Items      []audit.Row
			NextCursor string
			Total      int
		}
		json.Unmarshal(raw, &l)
		if st == 200 && l.Items == nil {
			t.Fatalf("items null: %s", raw)
		}
		return st, l.Items, l.NextCursor
	}
	st, rows, _ := get("?entityId=" + id)
	if st != 200 || len(rows) != 3 || rows[0].Action != "department.delete" || rows[1].Action != "department.update" || rows[2].Action != "department.create" {
		t.Fatalf("newest first: %d %+v", st, rows)
	}
	if a := rows[0].Actor; a == nil || a.ID != e.Fx.AdminA || a.Email == "" {
		t.Fatalf("actor: %+v", rows[0].Actor)
	}
	if rows[0].Before == nil || rows[0].After != nil || rows[2].Before != nil || rows[2].After == nil {
		t.Fatalf("before/after: %+v", rows)
	}
	if _, rows, _ = get("?entityType=term"); len(rows) != 0 {
		t.Fatalf("entityType=term: %d", len(rows))
	}
	if _, rows, _ = get("?entityType=department&actorUserId=" + e.Fx.AdminA); len(rows) != 3 {
		t.Fatalf("actor filter: %d", len(rows))
	}
	if _, rows, _ = get("?actorUserId=00000000-0000-4000-8000-000000000001"); len(rows) != 0 {
		t.Fatalf("other actor: %d", len(rows))
	}
	future, past := time.Now().Add(time.Hour).UTC().Format(time.RFC3339), time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	for q, want := range map[string]int{"from=" + past: 3, "from=" + future: 0, "to=" + future: 3, "to=" + past: 0} {
		if _, rows, _ = get("?entityId=" + id + "&" + url.PathEscape(q)); len(rows) != want {
			t.Fatalf("%s: %d want %d", q, len(rows), want)
		}
	}
	// pagination
	if tot := total(t, e, adm, "?entityId="+id+"&limit=1"); tot != 3 {
		t.Fatalf("total with limit: %d", tot)
	}
	if tot := total(t, e, adm, "?entityType=term"); tot != 0 {
		t.Fatalf("total entityType=term: %d", tot)
	}
	_, rows, next := get("?entityId=" + id + "&limit=2")
	if tot := total(t, e, adm, "?entityId="+id+"&limit=2&cursor="+next); tot != 3 {
		t.Fatalf("total ignores cursor: %d", tot)
	}
	_, rest, next2 := get("?entityId=" + id + "&limit=2&cursor=" + next)
	if len(rows) != 2 || next == "" || len(rest) != 1 || next2 != "" || rest[0].ID == rows[0].ID || rest[0].ID == rows[1].ID {
		t.Fatalf("paging: %d %d", len(rows), len(rest))
	}
	// validation
	for _, q := range []string{"?entityType=bogus", "?entityId=x", "?actorUserId=x", "?from=yesterday", "?to=2026-01-01", "?limit=x"} {
		if st, raw := e.do(adm, "GET", pathA+"/audit-log"+q, nil); st != 400 {
			t.Fatalf("%s: %d %s", q, st, raw)
		}
	}
	if st, _ := e.do(da, "GET", pathA+"/audit-log", nil); st != 403 {
		t.Fatalf("dept_admin: %d", st)
	}
	if st, _ := e.do(nil, "GET", pathA+"/audit-log", nil); st != 401 {
		t.Fatalf("anonymous: %d", st)
	}
}

func total(t *testing.T, e *env, cookie *http.Cookie, q string) int {
	t.Helper()
	_, raw := e.do(cookie, "GET", pathA+"/audit-log"+q, nil)
	var l struct{ Total *int }
	if err := json.Unmarshal(raw, &l); err != nil || l.Total == nil {
		t.Fatalf("total missing: %s", raw)
	}
	return *l.Total
}

func TestAuditLogCalendarFeed(t *testing.T) {
	e := newServer(t, regTenant)
	adm := e.adminCookie()
	id := "0b6f1c2e-5d4a-4f3b-9a51-7c2d8e9f0a11"
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return audit.Write(context.Background(), tx, e.Fx.UniversityA, audit.Entry{ActorUserID: e.Fx.AdminA, Action: "calendar_feed.create", EntityType: "calendar_feed", EntityID: id})
	})
	st, raw := e.do(adm, "GET", pathA+"/audit-log?entityType=calendar_feed", nil)
	if st != 200 || !strings.Contains(string(raw), `"calendar_feed.create"`) {
		t.Fatalf("filter: %d %s", st, raw)
	}
	if st, raw := e.do(adm, "GET", pathA+"/audit-log?entityType=term", nil); st != 200 || strings.Contains(string(raw), "calendar_feed") {
		t.Fatalf("other type: %d %s", st, raw)
	}
}
