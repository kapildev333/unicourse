// Package audit writes and lists the append-only audit log inside tenant transactions.
package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
)

// Entry is one change. ActorUserID is empty for CLI-written rows; Before and After may be nil.
type Entry struct {
	ActorUserID, Action, EntityType, EntityID string
	Before, After                             any
}

// Write inserts the entry in the caller's transaction, so it commits or rolls back with the change.
func Write(ctx context.Context, tx *sql.Tx, universityID string, e Entry) error {
	return WriteMany(ctx, tx, universityID, []Entry{e})
}

// writeChunk bounds one statement; each chunk is one round trip.
const writeChunk = 5000

// WriteMany inserts the entries in the caller's transaction, one statement per 5,000 entries.
func WriteMany(ctx context.Context, tx *sql.Tx, universityID string, es []Entry) error {
	for len(es) > 0 {
		n := min(len(es), writeChunk)
		if err := writeBatch(ctx, tx, universityID, es[:n]); err != nil {
			return err
		}
		es = es[n:]
	}
	return nil
}

func writeBatch(ctx context.Context, tx *sql.Tx, universityID string, es []Entry) error {
	actors, actions, types, ids := make([]string, len(es)), make([]string, len(es)), make([]string, len(es)), make([]string, len(es))
	befores, afters := make([]*string, len(es)), make([]*string, len(es))
	for i, e := range es {
		actors[i], actions[i], types[i], ids[i] = e.ActorUserID, e.Action, e.EntityType, e.EntityID
		var err error
		if befores[i], err = payload(e.Before); err != nil {
			return fmt.Errorf("audit before: %w", err)
		}
		if afters[i], err = payload(e.After); err != nil {
			return fmt.Errorf("audit after: %w", err)
		}
	}
	_, err := tx.ExecContext(ctx, `insert into audit_log (university_id, actor_user_id, action, entity_type, entity_id, before, after)
		select $1, nullif(a, '')::uuid, ac, et, eid::uuid, b::jsonb, af::jsonb
		from unnest($2::text[], $3::text[], $4::text[], $5::text[], $6::text[], $7::text[]) with ordinality as x(a, ac, et, eid, b, af, n)
		order by n`,
		universityID, actors, actions, types, ids, befores, afters)
	if err != nil {
		return fmt.Errorf("audit write: %w", err)
	}
	return nil
}

func payload(v any) (*string, error) {
	if v == nil {
		return nil, nil
	}
	b, err := json.Marshal(v)
	s := string(b)
	return &s, err
}

// Filter narrows List; zero values are ignored. From and To are inclusive.
type Filter struct {
	EntityType, EntityID, ActorUserID string
	From, To                          time.Time
}

type Row struct {
	ID          string          `json:"id"`
	ActorUserID string          `json:"actorUserId,omitempty"`
	Action      string          `json:"action"`
	EntityType  string          `json:"entityType"`
	EntityID    string          `json:"entityId"`
	Before      json.RawMessage `json:"before,omitempty"`
	After       json.RawMessage `json:"after,omitempty"`
	At          time.Time       `json:"at"`
}

// List returns rows newest first, keyset-paginated on (at, id).
func List(ctx context.Context, tx *sql.Tx, f Filter, p httpx.Page) (httpx.List[Row], error) {
	limit := p.Limit
	if limit <= 0 {
		limit = 50
	}
	q := `select id, actor_user_id, action, entity_type, entity_id, before, after, at from audit_log where true`
	var args []any
	add := func(cond string, v any) {
		args = append(args, v)
		q += fmt.Sprintf(" and "+cond, len(args))
	}
	if f.EntityType != "" {
		add("entity_type = $%d", f.EntityType)
	}
	if f.EntityID != "" {
		add("entity_id = $%d", f.EntityID)
	}
	if f.ActorUserID != "" {
		add("actor_user_id = $%d", f.ActorUserID)
	}
	if !f.From.IsZero() {
		add("at >= $%d", f.From)
	}
	if !f.To.IsZero() {
		add("at <= $%d", f.To)
	}
	if at, id, ok := p.After(); ok {
		args = append(args, at, id)
		q += fmt.Sprintf(" and (at, id) < ($%d, $%d)", len(args)-1, len(args))
	}
	args = append(args, limit+1)
	q += fmt.Sprintf(" order by at desc, id desc limit $%d", len(args))

	rows, err := tx.QueryContext(ctx, q, args...)
	if err != nil {
		return httpx.List[Row]{}, fmt.Errorf("audit list: %w", err)
	}
	defer rows.Close()
	out := httpx.List[Row]{Items: []Row{}}
	for rows.Next() {
		var r Row
		var actor sql.NullString
		var before, after []byte
		if err := rows.Scan(&r.ID, &actor, &r.Action, &r.EntityType, &r.EntityID, &before, &after, &r.At); err != nil {
			return httpx.List[Row]{}, fmt.Errorf("audit list: %w", err)
		}
		r.ActorUserID, r.Before, r.After = actor.String, before, after
		out.Items = append(out.Items, r)
	}
	if err := rows.Err(); err != nil {
		return httpx.List[Row]{}, fmt.Errorf("audit list: %w", err)
	}
	if len(out.Items) > limit {
		out.Items = out.Items[:limit]
		last := out.Items[limit-1]
		out.NextCursor = httpx.NextCursor(last.At, last.ID)
	}
	return out, nil
}
