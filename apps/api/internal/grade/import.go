package grade

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/kapildev333/unicourse/apps/api/internal/auth"
	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/csvimport"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

type scoreRef struct{ UserID, Points string }

// emailKey normalises for the duplicate key (raw lowercased when invalid).
func emailKey(raw string) string {
	if e, err := auth.NormalizeEmail(raw); err == nil {
		return e
	}
	return strings.ToLower(strings.TrimSpace(raw))
}

// importScores serves the preview (writes nothing) and the commit (re-classifies, then applies).
func importScores(commit bool) tenant.Handler {
	return func(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
		secID := r.PathValue("id")
		if err := sectionExists(r, tx, secID); err != nil {
			return 0, nil, err
		}
		a, err := loadAssessment(r, tx, secID, r.PathValue("assessmentId"), false)
		if err != nil {
			return 0, nil, err
		}
		if err := grade(r, tx, s, secID); err != nil {
			return 0, nil, err
		}
		spec := csvimport.Spec{Required: []string{"email", "points"},
			Key:     func(m map[string]string) string { return emailKey(m["email"]) },
			Actions: []csvimport.Action{csvimport.Add, csvimport.Update, csvimport.Exists, csvimport.Error}}
		recs, sum, form, err := csvimport.ReadUpload(r, spec)
		if err != nil {
			return 0, nil, err
		}
		p, err := csvimport.PreviewRows(r.Context(), tx, recs, sum, spec, scoreClassifier(a))
		if err != nil {
			return 0, nil, err
		}
		if commit {
			err = csvimport.Apply(r.Context(), tx, p, form, func(ctx context.Context, tx *sql.Tx, rows []csvimport.Row) error {
				in := make([]scoreIn, len(rows))
				for i, row := range rows {
					ref := row.Ref.(scoreRef)
					in[i] = scoreIn{ref.UserID, ref.Points}
				}
				return setScores(ctx, tx, s, a, in)
			})
			if err != nil {
				return 0, nil, err
			}
		}
		return http.StatusOK, p, nil
	}
}

func scoreClassifier(a Assessment) csvimport.Classifier {
	verr := func(msg string) csvimport.Verdict { return csvimport.Verdict{Action: csvimport.Error, Message: msg} }
	return func(ctx context.Context, tx *sql.Tx, recs []csvimport.Record) ([]csvimport.Verdict, error) {
		norm := make([]string, len(recs))
		emails := []string{}
		for i, rc := range recs {
			if e, err := auth.NormalizeEmail(rc.Data["email"]); err == nil {
				norm[i] = e
				emails = append(emails, e)
			}
		}
		pairs := func(q string, args ...any) (map[string]string, error) {
			rows, err := tx.QueryContext(ctx, q, args...)
			if err != nil {
				return nil, err
			}
			defer rows.Close()
			out := map[string]string{}
			for rows.Next() {
				var k, v string
				if err := rows.Scan(&k, &v); err != nil {
					return nil, err
				}
				out[k] = v
			}
			return out, rows.Err()
		}
		enrolled, err := pairs(`select u.email, u.id::text from enrollments e join users u on u.id = e.user_id where e.section_id = $1 and u.email = any($2::text[])`, a.SectionID, emails)
		if err != nil {
			return nil, err
		}
		// members (RLS limits memberships to this university) tell "not enrolled" from "unknown"
		members, err := pairs(`select u.email, u.id::text from memberships m join users u on u.id = m.user_id where u.email = any($1::text[])`, emails)
		if err != nil {
			return nil, err
		}
		ids := make([]string, 0, len(enrolled))
		for _, id := range enrolled {
			ids = append(ids, id)
		}
		current, err := pairs(`select user_id::text, points::text from scores where assessment_id = $1 and user_id = any($2::uuid[])`, a.ID, ids)
		if err != nil {
			return nil, err
		}
		out := make([]csvimport.Verdict, len(recs))
		for i, rc := range recs {
			e := norm[i]
			id, in := enrolled[e]
			_, member := members[e]
			var pts string
			var fe *httpx.FieldError
			if in {
				pts, fe = tenant.Decimal("points", json.Number(rc.Data["points"]), 2, "0", maxScore(a))
			}
			switch {
			case e == "":
				out[i] = verr("invalid email")
			case !in && !member:
				out[i] = verr("unknown email")
			case !in:
				out[i] = verr("not enrolled in this section")
			case fe != nil:
				out[i] = verr("points " + fe.Message)
			default:
				old, has := current[id]
				switch {
				case !has:
					out[i] = csvimport.Verdict{Action: csvimport.Add, Ref: scoreRef{id, pts}}
				case rat(json.Number(old)).Cmp(rat(json.Number(pts))) == 0:
					out[i] = csvimport.Verdict{Action: csvimport.Exists}
				default:
					out[i] = csvimport.Verdict{Action: csvimport.Update, Message: round2(rat(json.Number(old))).String() + " → " + pts, Ref: scoreRef{id, pts}}
				}
			}
		}
		return out, nil
	}
}
