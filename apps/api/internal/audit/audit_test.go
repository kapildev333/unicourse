package audit_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/audit"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
	"github.com/kapildev333/unicourse/apps/api/internal/testdb"
)

var ctx = context.Background()

const entity = "11111111-1111-1111-1111-111111111111"

func write(t *testing.T, db *sql.DB, uni string, e audit.Entry) {
	t.Helper()
	if err := pg.WithTx(ctx, db, uni, func(tx *sql.Tx) error { return audit.Write(ctx, tx, uni, e) }); err != nil {
		t.Fatal(err)
	}
}

func list(t *testing.T, db *sql.DB, uni string, f audit.Filter, p httpx.Page) (out httpx.List[audit.Row]) {
	t.Helper()
	err := pg.WithTx(ctx, db, uni, func(tx *sql.Tx) (err error) { out, err = audit.List(ctx, tx, f, p); return })
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestWriteAndList(t *testing.T) {
	db := testdb.Open(t)
	f := testdb.Seed(t, db)
	write(t, db, f.UniversityA, audit.Entry{ActorUserID: f.AdminA, Action: "section.update", EntityType: "section", EntityID: entity,
		Before: map[string]string{"number": "001"}, After: map[string]string{"number": "002"}})
	write(t, db, f.UniversityA, audit.Entry{Action: "course.create", EntityType: "course", EntityID: entity}) // CLI write: no actor, no payloads

	got := list(t, db, f.UniversityA, audit.Filter{EntityType: "section"}, httpx.Page{Limit: 50})
	if len(got.Items) != 1 || got.NextCursor != "" {
		t.Fatalf("got %+v", got)
	}
	r := got.Items[0]
	if r.ActorUserID != f.AdminA || r.Action != "section.update" || r.EntityID != entity || r.At.IsZero() || r.ID == "" ||
		string(r.Before) != `{"number": "001"}` || string(r.After) != `{"number": "002"}` {
		t.Fatalf("row %+v before %s after %s", r, r.Before, r.After)
	}

	// no actor and no payloads marshal without those keys
	all := list(t, db, f.UniversityA, audit.Filter{EntityType: "course"}, httpx.Page{Limit: 50})
	b, _ := json.Marshal(all.Items[0])
	var m map[string]any
	json.Unmarshal(b, &m)
	for _, k := range []string{"actorUserId", "before", "after"} {
		if _, ok := m[k]; ok {
			t.Errorf("%s present in %s", k, b)
		}
	}

	// B sees nothing and gets [] not null
	none := list(t, db, f.UniversityB, audit.Filter{}, httpx.Page{Limit: 50})
	if b, _ := json.Marshal(none); string(b) != `{"items":[]}` {
		t.Fatalf("B: %s", b)
	}
	if got := list(t, db, f.UniversityA, audit.Filter{EntityID: entity, ActorUserID: f.AdminA}, httpx.Page{Limit: 50}); len(got.Items) != 1 {
		t.Fatalf("entity+actor filter: %+v", got)
	}
}

func TestPaginationNewestFirstAndRange(t *testing.T) {
	db := testdb.Open(t)
	f := testdb.Seed(t, db)
	for _, a := range []string{"one", "two", "three"} {
		write(t, db, f.UniversityA, audit.Entry{Action: a, EntityType: "section", EntityID: entity})
		time.Sleep(5 * time.Millisecond)
	}
	p1 := list(t, db, f.UniversityA, audit.Filter{}, httpx.Page{Limit: 2})
	if len(p1.Items) != 2 || p1.NextCursor == "" || p1.Items[0].Action != "three" || p1.Items[1].Action != "two" {
		t.Fatalf("page 1: %+v", p1)
	}
	p2 := list(t, db, f.UniversityA, audit.Filter{}, httpx.Page{Limit: 2, Cursor: p1.NextCursor})
	if len(p2.Items) != 1 || p2.NextCursor != "" || p2.Items[0].Action != "one" {
		t.Fatalf("page 2: %+v", p2)
	}

	mid := p1.Items[1].At // "two"
	if got := list(t, db, f.UniversityA, audit.Filter{From: mid}, httpx.Page{Limit: 50}); len(got.Items) != 2 {
		t.Errorf("from: %d items, want 2", len(got.Items))
	}
	if got := list(t, db, f.UniversityA, audit.Filter{To: mid}, httpx.Page{Limit: 50}); len(got.Items) != 2 {
		t.Errorf("to: %d items, want 2", len(got.Items))
	}
	if got := list(t, db, f.UniversityA, audit.Filter{From: mid, To: mid}, httpx.Page{Limit: 50}); len(got.Items) != 1 {
		t.Errorf("from=to: %d items, want 1", len(got.Items))
	}
}

func TestWriteWithoutTenantFails(t *testing.T) {
	db := testdb.Open(t)
	f := testdb.Seed(t, db)
	err := pg.WithTx(ctx, db, f.UniversityA, func(tx *sql.Tx) error {
		return audit.Write(ctx, tx, f.UniversityB, audit.Entry{Action: "x", EntityType: "section", EntityID: entity})
	})
	if err == nil {
		t.Fatal("cross-tenant audit write accepted")
	}
}

func TestWriteStructPayloads(t *testing.T) {
	db := testdb.Open(t)
	f := testdb.Seed(t, db)
	type dept struct {
		Code string `json:"code"`
		N    int    `json:"n"`
	}
	write(t, db, f.UniversityA, audit.Entry{Action: "department.create", EntityType: "department", EntityID: entity, After: dept{"CS", 3}})
	got := list(t, db, f.UniversityA, audit.Filter{}, httpx.Page{Limit: 50}).Items[0]
	if string(got.After) != `{"n": 3, "code": "CS"}` || got.Before != nil {
		t.Fatalf("before %q after %q", got.Before, got.After)
	}
	var sqlNull bool
	pg.WithTx(ctx, db, f.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select before is null from audit_log`).Scan(&sqlNull)
	})
	if !sqlNull {
		t.Error("nil Before must be SQL NULL, not JSON null")
	}
}

// Rows sharing one `at` across a page boundary must be neither skipped nor repeated.
func TestPaginationTiedAt(t *testing.T) {
	db := testdb.Open(t)
	f := testdb.Seed(t, db)
	err := pg.WithTx(ctx, db, f.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`insert into audit_log (university_id, action, entity_type, entity_id, at)
			select $1, 'a' || n, 'section', $2, '2026-01-01T00:00:00Z' from generate_series(1, 5) n`, f.UniversityA, entity)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	cursor := ""
	for range 5 {
		p := list(t, db, f.UniversityA, audit.Filter{}, httpx.Page{Limit: 2, Cursor: cursor})
		for _, r := range p.Items {
			if seen[r.ID] {
				t.Fatalf("row %s repeated", r.ID)
			}
			seen[r.ID] = true
		}
		if cursor = p.NextCursor; cursor == "" {
			break
		}
	}
	if len(seen) != 5 {
		t.Fatalf("saw %d rows, want 5", len(seen))
	}
}
