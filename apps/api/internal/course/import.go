package course

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/kapildev333/unicourse/apps/api/internal/audit"
	"github.com/kapildev333/unicourse/apps/api/internal/auth"
	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/csvimport"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

// importer authorises the request (before the upload is read) and returns the import's pieces.
type importer func(d tenant.Deps, r *http.Request, tx *sql.Tx, s authz.Scope) (csvimport.Spec, csvimport.Classifier, csvimport.Applier, error)

// importHandler serves both the preview (writes nothing) and the commit (re-classifies, then applies).
func importHandler(d tenant.Deps, commit bool, mk importer) tenant.Handler {
	return func(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
		spec, cl, ap, err := mk(d, r, tx, s)
		if err != nil {
			return 0, nil, err
		}
		recs, sum, form, err := csvimport.ReadUpload(r, spec)
		if err != nil {
			return 0, nil, err
		}
		p, err := csvimport.PreviewRows(r.Context(), tx, recs, sum, spec, cl)
		if err != nil {
			return 0, nil, err
		}
		if commit {
			if err := csvimport.Apply(r.Context(), tx, p, form, ap); err != nil {
				return 0, nil, err
			}
		}
		return http.StatusOK, p, nil
	}
}

// emailKey normalises for the duplicate key (raw lowercased when invalid).
func emailKey(raw string) string {
	if e, err := auth.NormalizeEmail(raw); err == nil {
		return e
	}
	return strings.ToLower(strings.TrimSpace(raw))
}

func verr(msg string) csvimport.Verdict {
	return csvimport.Verdict{Action: csvimport.Error, Message: msg}
}

func idSet(ctx context.Context, tx *sql.Tx, q string, args ...any) (map[string]bool, error) {
	rows, err := tx.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		out[k] = true
	}
	return out, rows.Err()
}

func strMap(ctx context.Context, tx *sql.Tx, q string, args ...any) (map[string][]string, error) {
	rows, err := tx.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]string{}
	for rows.Next() {
		var k string
		var v [2]string
		if err := rows.Scan(&k, &v[0], &v[1]); err != nil {
			return nil, err
		}
		out[k] = v[:]
	}
	return out, rows.Err()
}

// ---- 8.1 enrollments ----

type enrollRef struct{ UserID, Email, InvitationID string }

func enrollmentsImport(d tenant.Deps, r *http.Request, tx *sql.Tx, s authz.Scope) (csvimport.Spec, csvimport.Classifier, csvimport.Applier, error) {
	secID := r.PathValue("id")
	if err := sectionExists(r, tx, secID); err != nil {
		return csvimport.Spec{}, nil, nil, err
	}
	if err := allow(authz.CanManageSection(r.Context(), tx, s, secID)); err != nil {
		return csvimport.Spec{}, nil, nil, err
	}
	spec := csvimport.Spec{Required: []string{"email"}, Optional: []string{"displayname"},
		Key: func(m map[string]string) string { return emailKey(m["email"]) }}
	return spec, enrollmentClassifier(s, secID), enrollmentApplier(d, s, secID), nil
}

func enrollmentClassifier(s authz.Scope, secID string) csvimport.Classifier {
	return func(ctx context.Context, tx *sql.Tx, recs []csvimport.Record) ([]csvimport.Verdict, error) {
		norm := make([]string, len(recs))
		valid := []string{}
		for i, rc := range recs {
			if e, err := auth.NormalizeEmail(rc.Data["email"]); err == nil {
				norm[i] = e
				valid = append(valid, e)
			}
		}
		states, err := resolveEmails(ctx, tx, valid)
		if err != nil {
			return nil, err
		}
		users, invs := []string{}, []string{}
		for _, st := range states {
			if st.UserID != "" {
				users = append(users, st.UserID)
			}
			if st.InvitationID != "" {
				invs = append(invs, st.InvitationID)
			}
		}
		enrolled, err := idSet(ctx, tx, `select user_id::text from enrollments where section_id = $1 and user_id = any($2::uuid[])`, secID, users)
		if err != nil {
			return nil, err
		}
		pending, err := idSet(ctx, tx, `select invitation_id::text from pending_enrollments where section_id = $1 and invitation_id = any($2::uuid[])`, secID, invs)
		if err != nil {
			return nil, err
		}
		var raw []byte
		if err := tx.QueryRowContext(ctx, `select to_json(allowed_email_domains) from universities where id = $1`, s.UniversityID).Scan(&raw); err != nil {
			return nil, err
		}
		var domains []string
		if err := json.Unmarshal(raw, &domains); err != nil {
			return nil, err
		}
		out := make([]csvimport.Verdict, len(recs))
		for i := range recs {
			e := norm[i]
			st := states[e]
			switch {
			case e == "":
				out[i] = verr("invalid email")
			case st.Status == "active":
				if enrolled[st.UserID] {
					out[i] = csvimport.Verdict{Action: csvimport.Exists}
				} else {
					out[i] = csvimport.Verdict{Action: csvimport.Add, Ref: enrollRef{UserID: st.UserID, Email: e}}
				}
			case st.Status != "":
				out[i] = verr("membership is not active")
			case st.InvitationID != "":
				if pending[st.InvitationID] {
					out[i] = csvimport.Verdict{Action: csvimport.Exists}
				} else {
					out[i] = csvimport.Verdict{Action: csvimport.Invite, Ref: enrollRef{Email: e, InvitationID: st.InvitationID}}
				}
			case slices.Contains(domains, e[strings.LastIndex(e, "@")+1:]):
				out[i] = csvimport.Verdict{Action: csvimport.Invite, Ref: enrollRef{Email: e}}
			default:
				out[i] = verr("domain not allowed")
			}
		}
		return out, nil
	}
}

func enrollmentApplier(d tenant.Deps, s authz.Scope, secID string) csvimport.Applier {
	return func(ctx context.Context, tx *sql.Tx, rows []csvimport.Row) error {
		var users, newEmails, invIDs []string
		emailOf := map[string]string{}
		for _, row := range rows {
			ref := row.Ref.(enrollRef)
			switch {
			case ref.UserID != "":
				users = append(users, ref.UserID)
				emailOf[ref.UserID] = ref.Email
			case ref.InvitationID != "":
				invIDs = append(invIDs, ref.InvitationID)
			default:
				newEmails = append(newEmails, ref.Email)
			}
		}
		if len(users) > 0 {
			res, err := tx.QueryContext(ctx, `insert into enrollments (university_id, section_id, user_id)
				select $1, $2, u from unnest($3::uuid[]) u on conflict (university_id, section_id, user_id) do nothing returning id, user_id`, s.UniversityID, secID, users)
			if err != nil {
				return err
			}
			var es []audit.Entry
			for res.Next() {
				var id, uid string
				if err := res.Scan(&id, &uid); err != nil {
					res.Close()
					return err
				}
				es = append(es, audit.Entry{ActorUserID: s.UserID, Action: "enrollment.create", EntityType: "enrollment", EntityID: id,
					After: enrollmentAudit{SectionID: secID, UserID: uid, Email: emailOf[uid]}})
			}
			res.Close()
			if err := res.Err(); err != nil {
				return err
			}
			if err := audit.WriteMany(ctx, tx, s.UniversityID, es); err != nil {
				return err
			}
		}
		invs, err := tenant.CreateInvitations(ctx, tx, s.UniversityID, s.UserID, newEmails, authz.Member, d.Mailer, d.BaseURL)
		if err != nil {
			return err
		}
		for _, v := range invs {
			invIDs = append(invIDs, v.ID)
		}
		if len(invIDs) == 0 {
			return nil
		}
		_, err = tx.ExecContext(ctx, `insert into pending_enrollments (university_id, invitation_id, section_id)
			select $1, i, $2 from unnest($3::uuid[]) i on conflict do nothing`, s.UniversityID, secID, invIDs)
		return err
	}
}

// ---- 8.2 courses ----

func coursesImport(d tenant.Deps, _ *http.Request, _ *sql.Tx, s authz.Scope) (csvimport.Spec, csvimport.Classifier, csvimport.Applier, error) {
	if plainMember(s) {
		return csvimport.Spec{}, nil, nil, httpx.Forbidden()
	}
	spec := csvimport.Spec{Required: []string{"departmentcode", "code", "title"},
		Key: func(m map[string]string) string { return strings.TrimSpace(m["code"]) }}
	return spec, courseClassifier(s), courseApplier(s), nil
}

type courseRef struct{ DeptID, Code, Title string }

func courseClassifier(s authz.Scope) csvimport.Classifier {
	return func(ctx context.Context, tx *sql.Tx, recs []csvimport.Record) ([]csvimport.Verdict, error) {
		depts, err := strMap(ctx, tx, `select code, id::text, '' from departments`)
		if err != nil {
			return nil, err
		}
		codes := make([]string, len(recs))
		for i, rc := range recs {
			codes[i] = rc.Data["code"]
		}
		have, err := idSet(ctx, tx, `select code from courses where code = any($1::text[])`, codes)
		if err != nil {
			return nil, err
		}
		out := make([]csvimport.Verdict, len(recs))
		for i, rc := range recs {
			code, title, dc := rc.Data["code"], rc.Data["title"], rc.Data["departmentcode"]
			d, known := depts[dc]
			switch {
			case code == "":
				out[i] = verr("code is required")
			case title == "":
				out[i] = verr("title is required")
			case utf8.RuneCountInString(code) > 20:
				out[i] = verr("code must be at most 20 characters")
			case utf8.RuneCountInString(title) > 200:
				out[i] = verr("title must be at most 200 characters")
			case !known:
				out[i] = verr("unknown department code " + dc)
			case !authz.CanManageDepartment(s, d[0]):
				out[i] = verr("department " + dc + " is not managed by you")
			case have[code]:
				out[i] = csvimport.Verdict{Action: csvimport.Exists}
			default:
				out[i] = csvimport.Verdict{Action: csvimport.Add, Ref: courseRef{d[0], code, title}}
			}
		}
		return out, nil
	}
}

func courseApplier(s authz.Scope) csvimport.Applier {
	return func(ctx context.Context, tx *sql.Tx, rows []csvimport.Row) error {
		var ds, cs, ts []string
		for _, row := range rows {
			ref := row.Ref.(courseRef)
			ds, cs, ts = append(ds, ref.DeptID), append(cs, ref.Code), append(ts, ref.Title)
		}
		res, err := tx.QueryContext(ctx, `insert into courses (university_id, department_id, code, title)
			select $1, d, c, t from unnest($2::uuid[], $3::text[], $4::text[]) as x(d, c, t) returning id, department_id, code, title`, s.UniversityID, ds, cs, ts)
		if err != nil {
			return courseWriteErr(err)
		}
		var es []audit.Entry
		for res.Next() {
			var c Course
			if err := res.Scan(&c.ID, &c.DepartmentID, &c.Code, &c.Title); err != nil {
				res.Close()
				return err
			}
			es = append(es, audit.Entry{ActorUserID: s.UserID, Action: "course.create", EntityType: "course", EntityID: c.ID, After: c})
		}
		res.Close()
		if err := res.Err(); err != nil {
			return courseWriteErr(err)
		}
		return audit.WriteMany(ctx, tx, s.UniversityID, es)
	}
}

// ---- 8.3 sections ----

type staffAdd struct{ UserID, Email, Role string }
type pendAdd struct{ Email, InvitationID, Role string }
type sectionRef struct {
	CourseID, TermID, Number, SectionID string
	Staff                               []staffAdd
	Pending                             []pendAdd
}

func sectionsImport(d tenant.Deps, _ *http.Request, _ *sql.Tx, s authz.Scope) (csvimport.Spec, csvimport.Classifier, csvimport.Applier, error) {
	if plainMember(s) {
		return csvimport.Spec{}, nil, nil, httpx.Forbidden()
	}
	spec := csvimport.Spec{Required: []string{"coursecode", "termname", "number"}, Optional: []string{"instructoremails", "taemails"},
		Key: func(m map[string]string) string {
			return strings.TrimSpace(m["coursecode"]) + "\x00" + strings.TrimSpace(m["termname"]) + "\x00" + strings.TrimSpace(m["number"])
		}}
	return spec, sectionClassifier(s), sectionApplier(d, s), nil
}

// splitEmails returns the normalised, deduplicated emails of a cell, or the first invalid raw value.
func splitEmails(cell string) (out []string, bad string) {
	for _, p := range strings.Split(cell, ";") {
		if p = strings.TrimSpace(p); p == "" {
			continue
		}
		e, err := auth.NormalizeEmail(p)
		if err != nil {
			return nil, p
		}
		if !slices.Contains(out, e) {
			out = append(out, e)
		}
	}
	return out, ""
}

func sectionClassifier(s authz.Scope) csvimport.Classifier {
	return func(ctx context.Context, tx *sql.Tx, recs []csvimport.Record) ([]csvimport.Verdict, error) {
		codes, terms, all := []string{}, []string{}, []string{}
		for _, rc := range recs {
			codes, terms = append(codes, rc.Data["coursecode"]), append(terms, rc.Data["termname"])
			for _, c := range []string{"instructoremails", "taemails"} {
				es, _ := splitEmails(rc.Data[c])
				all = append(all, es...)
			}
		}
		courses, err := strMap(ctx, tx, `select code, id::text, department_id::text from courses where code = any($1::text[])`, codes)
		if err != nil {
			return nil, err
		}
		termIDs, err := strMap(ctx, tx, `select name, id::text, '' from terms where name = any($1::text[])`, terms)
		if err != nil {
			return nil, err
		}
		var cids, tids, nums []string
		for _, rc := range recs {
			c, t := courses[rc.Data["coursecode"]], termIDs[rc.Data["termname"]]
			if c != nil && t != nil && rc.Data["number"] != "" {
				cids, tids, nums = append(cids, c[0]), append(tids, t[0]), append(nums, rc.Data["number"])
			}
		}
		existing := map[string]string{} // course|term|number -> section id
		secIDs := []string{}
		rows, err := tx.QueryContext(ctx, `select s.id::text, s.course_id::text, s.term_id::text, s.number from sections s
			join unnest($1::uuid[], $2::uuid[], $3::text[]) as x(c, t, n) on s.course_id = x.c and s.term_id = x.t and s.number = x.n`, cids, tids, nums)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var id, c, t, n string
			if err := rows.Scan(&id, &c, &t, &n); err != nil {
				rows.Close()
				return nil, err
			}
			existing[c+"|"+t+"|"+n] = id
			secIDs = append(secIDs, id)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
		staffSet, err := idSet(ctx, tx, `select section_id::text || '|' || user_id::text from section_staff where section_id = any($1::uuid[])`, secIDs)
		if err != nil {
			return nil, err
		}
		pendSet, err := idSet(ctx, tx, `select section_id::text || '|' || invitation_id::text from pending_section_staff where section_id = any($1::uuid[])`, secIDs)
		if err != nil {
			return nil, err
		}
		states, err := resolveEmails(ctx, tx, all)
		if err != nil {
			return nil, err
		}
		out := make([]csvimport.Verdict, len(recs))
		for i, rc := range recs {
			out[i] = classifySection(s, rc, courses, termIDs, existing, staffSet, pendSet, states)
		}
		return out, nil
	}
}

func classifySection(s authz.Scope, rc csvimport.Record, courses, terms map[string][]string, existing map[string]string,
	staffSet, pendSet map[string]bool, states map[string]emailState) csvimport.Verdict {
	d := rc.Data
	c, t := courses[d["coursecode"]], terms[d["termname"]]
	switch {
	case d["number"] == "":
		return verr("number is required")
	case utf8.RuneCountInString(d["number"]) > 20:
		return verr("number must be at most 20 characters")
	case c == nil:
		return verr("unknown course code " + d["coursecode"])
	case t == nil:
		return verr("unknown term " + d["termname"])
	case !authz.CanManageDepartment(s, c[1]):
		return verr("course " + d["coursecode"] + " is not in a department you manage")
	}
	lists := [2][]string{}
	for i, col := range []string{"instructoremails", "taemails"} {
		es, bad := splitEmails(d[col])
		if bad != "" {
			return verr("invalid email " + bad)
		}
		lists[i] = es
	}
	for _, e := range lists[0] {
		if slices.Contains(lists[1], e) {
			return verr(e + " listed as both instructor and ta")
		}
	}
	ref := sectionRef{CourseID: c[0], TermID: t[0], Number: d["number"]}
	ref.SectionID = existing[ref.CourseID+"|"+ref.TermID+"|"+ref.Number]
	invite, addStaff := false, false
	for i, role := range []string{"instructor", "ta"} {
		for _, e := range lists[i] {
			st := states[e]
			switch {
			case st.Status == "active":
				if !staffSet[ref.SectionID+"|"+st.UserID] {
					ref.Staff = append(ref.Staff, staffAdd{st.UserID, e, role})
					addStaff = true
				}
			case st.Status != "":
				return verr(e + ": membership is not active")
			default:
				if st.InvitationID == "" || !pendSet[ref.SectionID+"|"+st.InvitationID] {
					ref.Pending = append(ref.Pending, pendAdd{e, st.InvitationID, role})
					invite = true
				}
			}
		}
	}
	switch {
	case invite:
		return csvimport.Verdict{Action: csvimport.Invite, Ref: ref}
	case ref.SectionID == "" || addStaff:
		return csvimport.Verdict{Action: csvimport.Add, Ref: ref}
	}
	return csvimport.Verdict{Action: csvimport.Exists}
}

func sectionApplier(d tenant.Deps, s authz.Scope) csvimport.Applier {
	return func(ctx context.Context, tx *sql.Tx, rows []csvimport.Row) error {
		var cids, tids, nums []string
		secOf := map[string]string{} // course|term|number -> id
		for _, row := range rows {
			ref := row.Ref.(sectionRef)
			if ref.SectionID == "" {
				cids, tids, nums = append(cids, ref.CourseID), append(tids, ref.TermID), append(nums, ref.Number)
			} else {
				secOf[ref.CourseID+"|"+ref.TermID+"|"+ref.Number] = ref.SectionID
			}
		}
		if len(cids) > 0 {
			res, err := tx.QueryContext(ctx, `insert into sections (university_id, course_id, term_id, number)
				select $1, c, t, n from unnest($2::uuid[], $3::uuid[], $4::text[]) as x(c, t, n) returning id, course_id, term_id, number`, s.UniversityID, cids, tids, nums)
			if err != nil {
				return sectionWriteErr(err)
			}
			var es []audit.Entry
			for res.Next() {
				var a sectionAudit
				if err := res.Scan(&a.ID, &a.CourseID, &a.TermID, &a.Number); err != nil {
					res.Close()
					return err
				}
				secOf[a.CourseID+"|"+a.TermID+"|"+a.Number] = a.ID
				es = append(es, audit.Entry{ActorUserID: s.UserID, Action: "section.create", EntityType: "section", EntityID: a.ID, After: a})
			}
			res.Close()
			if err := res.Err(); err != nil {
				return sectionWriteErr(err)
			}
			if err := audit.WriteMany(ctx, tx, s.UniversityID, es); err != nil {
				return err
			}
		}
		var ssec, susr, srole, psec, pinv, prole []string
		emailOf := map[string]string{}
		newSet := map[string]bool{}
		type pend struct{ sec, email, inv, role string }
		var pends []pend
		for _, row := range rows {
			ref := row.Ref.(sectionRef)
			sec := secOf[ref.CourseID+"|"+ref.TermID+"|"+ref.Number]
			for _, st := range ref.Staff {
				ssec, susr, srole = append(ssec, sec), append(susr, st.UserID), append(srole, st.Role)
				emailOf[st.UserID] = st.Email
			}
			for _, p := range ref.Pending {
				pends = append(pends, pend{sec, p.Email, p.InvitationID, p.Role})
				if p.InvitationID == "" {
					newSet[p.Email] = true
				}
			}
		}
		if len(ssec) > 0 {
			res, err := tx.QueryContext(ctx, `insert into section_staff (university_id, section_id, user_id, role)
				select $1, a, b, c from unnest($2::uuid[], $3::uuid[], $4::text[]) as x(a, b, c)
				on conflict (university_id, section_id, user_id) do nothing returning id, section_id, user_id, role`, s.UniversityID, ssec, susr, srole)
			if err != nil {
				return err
			}
			var es []audit.Entry
			for res.Next() {
				var id string
				a := staffAudit{}
				if err := res.Scan(&id, &a.SectionID, &a.UserID, &a.Role); err != nil {
					res.Close()
					return err
				}
				a.Email = emailOf[a.UserID]
				es = append(es, audit.Entry{ActorUserID: s.UserID, Action: "staff.create", EntityType: "section_staff", EntityID: id, After: a})
			}
			res.Close()
			if err := res.Err(); err != nil {
				return err
			}
			if err := audit.WriteMany(ctx, tx, s.UniversityID, es); err != nil {
				return err
			}
		}
		fresh := make([]string, 0, len(newSet))
		for e := range newSet {
			fresh = append(fresh, e)
		}
		sort.Strings(fresh)
		invs, err := tenant.CreateInvitations(ctx, tx, s.UniversityID, s.UserID, fresh, authz.Member, d.Mailer, d.BaseURL)
		if err != nil {
			return err
		}
		invOf := map[string]string{}
		for _, v := range invs {
			invOf[v.Email] = v.ID
		}
		for _, p := range pends {
			id := p.inv
			if id == "" {
				id = invOf[p.email]
			}
			psec, pinv, prole = append(psec, p.sec), append(pinv, id), append(prole, p.role)
		}
		if len(psec) == 0 {
			return nil
		}
		_, err = tx.ExecContext(ctx, `insert into pending_section_staff (university_id, invitation_id, section_id, role)
			select $1, a, b, c from unnest($2::uuid[], $3::uuid[], $4::text[]) as x(a, b, c) on conflict do nothing`, s.UniversityID, pinv, psec, prole)
		return err
	}
}
