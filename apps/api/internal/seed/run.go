package seed

import (
	"bytes"
	"cmp"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"slices"
	"strings"

	"github.com/kapildev333/unicourse/apps/api/internal/app"
	"github.com/kapildev333/unicourse/apps/api/internal/auth"
	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/blob"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/outbox"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

var ErrExists = errors.New("university already exists")

type Options struct {
	Reset   bool
	BaseURL string   // UNICOURSE_BASE_URL, required: the router's invitation URLs (in mail that is never queued)
	Files   blob.Dir // UNICOURSE_FILES_DIR, from blob.New
}

type Report struct {
	UniversityID, Name, Slug string
	Personas                 []Persona // one per people entry, fixture order
	Warnings                 []string  // problems removing the files of a reset university
}
type Persona struct{ Role, Name, Email, Sees string }

// apiError is a non-2xx answer from the in-process router.
type apiError struct {
	What, As string
	Status   int
	Body     []byte
}

func (e *apiError) Error() string {
	var env struct {
		Error struct {
			Code, Message string
			Details       []httpx.FieldError
		}
	}
	msg := ""
	if json.Unmarshal(e.Body, &env) == nil && env.Error.Code != "" {
		msg = fmt.Sprintf("%s: %s", env.Error.Code, env.Error.Message)
		if len(env.Error.Details) > 0 {
			var d []string
			for _, f := range env.Error.Details {
				d = append(d, f.Field+": "+f.Message)
			}
			msg += " (" + strings.Join(d, "; ") + ")"
		}
	} else {
		msg = strings.TrimSpace(string(e.Body))
		if len(msg) > 500 {
			msg = msg[:500]
		}
	}
	return fmt.Sprintf("%s as %s: HTTP %d %s", e.What, e.As, e.Status, msg)
}

type writer struct {
	ctx    context.Context
	slug   string
	h      http.Handler
	store  *auth.SessionStore
	users  map[string]string // email -> user id
	tokens map[string]string // email -> session token
	depts  map[string]string // code -> id
	terms  map[string]string // name -> id
	names  map[string]string // email -> display name
}

// call makes one request as the persona with that email (session minted on first use) and decodes a 2xx
// JSON body into out (nil: ignore).
func (w *writer) call(what, as, method, path, contentType string, body io.Reader, out any) error {
	tok, ok := w.tokens[as]
	if !ok {
		var err error
		if tok, err = w.store.Create(w.ctx, w.users[as]); err != nil {
			return fmt.Errorf("sessions: %w", err)
		}
		w.tokens[as] = tok
	}
	req := httptest.NewRequestWithContext(w.ctx, method, "/api/v1/universities/"+w.slug+path, body)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.AddCookie(&http.Cookie{Name: auth.SessionCookie, Value: tok})
	rec := httptest.NewRecorder()
	w.h.ServeHTTP(rec, req)
	if rec.Code >= 300 {
		return &apiError{What: what, As: as, Status: rec.Code, Body: rec.Body.Bytes()}
	}
	if out != nil {
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			return fmt.Errorf("%s as %s: undecodable answer: %w", what, as, err)
		}
	}
	return nil
}

func (w *writer) post(what, as, method, path string, v, out any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return w.call(what, as, method, path, "application/json", bytes.NewReader(b), out)
}

type part struct {
	name, filename, ctype string
	data                  []byte
}

func (w *writer) upload(what, as, path string, parts []part, out any) error {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for _, p := range parts {
		h := textproto.MIMEHeader{}
		cd := fmt.Sprintf(`form-data; name=%q`, p.name)
		if p.filename != "" {
			cd += fmt.Sprintf(`; filename=%q`, p.filename)
			h.Set("Content-Type", p.ctype)
		}
		h.Set("Content-Disposition", cd)
		pw, err := mw.CreatePart(h)
		if err != nil {
			return err
		}
		pw.Write(p.data)
	}
	if err := mw.Close(); err != nil {
		return err
	}
	return w.call(what, as, http.MethodPost, path, mw.FormDataContentType(), &buf, out)
}

type idOnly struct {
	ID string `json:"id"`
}

// Run seeds f into db (the owner pool). ErrExists when the slug exists and !o.Reset.
func Run(ctx context.Context, db *sql.DB, f Fixture, o Options) (rep Report, err error) {
	if err := pg.CheckMigrated(ctx, db); err != nil {
		return rep, err
	}
	slug := f.University.Slug
	warnings, err := resetSlug(ctx, db, slug, o)
	if err != nil {
		return rep, err
	}
	rep = Report{Name: f.University.Name, Slug: slug, Warnings: warnings}
	// Past this point something may be left behind on failure.
	var uni string
	err = pg.WithPlatformTx(ctx, db, func(tx *sql.Tx) error {
		if err := tx.QueryRowContext(ctx,
			`insert into universities (name, slug, allowed_email_domains, timezone, locale) values ($1, $2, $3, $4, $5) returning id`,
			f.University.Name, slug, []string{slug + ".edu"}, f.University.Timezone, f.University.Locale).Scan(&uni); err != nil {
			return pg.MapError(err)
		}
		if err := pg.SetUniversity(ctx, tx, uni); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `select seed_us_grade_scale($1)`, uni)
		return err
	})
	if errors.Is(err, pg.ErrConflict) {
		return rep, ErrExists
	}
	if err != nil {
		return rep, fmt.Errorf("university: %w", err)
	}
	rep.UniversityID = uni
	rep.personas(f)

	store := &auth.SessionStore{DB: db}
	w := &writer{ctx: ctx, slug: slug, store: store,
		h:     httpx.RequestIDMiddleware(httpx.Recover(app.NewRouter(app.Deps{Auth: auth.Deps{DB: db, Sessions: store}, BaseURL: o.BaseURL, Files: o.Files}))),
		users: map[string]string{}, tokens: map[string]string{}, depts: map[string]string{}, terms: map[string]string{}, names: map[string]string{}}
	defer func() {
		for _, tok := range w.tokens {
			if rerr := store.Revoke(context.WithoutCancel(ctx), tok); rerr != nil {
				err = errors.Join(err, fmt.Errorf("sessions: %w", rerr))
			}
		}
	}()
	if err := w.seed(db, uni, f, o); err != nil {
		return rep, fmt.Errorf("%w; %s is partly seeded, rerun with --reset", err, slug)
	}
	return rep, nil
}

// resetSlug handles an existing university with this slug: ErrExists, or with Reset the delete and its files.
func resetSlug(ctx context.Context, db *sql.DB, slug string, o Options) (warnings []string, err error) {
	var old string
	err = pg.WithPlatformTx(ctx, db, func(tx *sql.Tx) error {
		switch err := tx.QueryRowContext(ctx, `select id from universities where slug = $1`, slug).Scan(&old); {
		case errors.Is(err, sql.ErrNoRows):
			return nil
		case err != nil:
			return err
		case !o.Reset:
			return ErrExists
		}
		_, err := tx.ExecContext(ctx, `delete from universities where id = $1`, old)
		return err
	})
	if err != nil || old == "" {
		return nil, err
	}
	if err := o.Files.RemoveUniversity(old); err != nil {
		warnings = append(warnings, fmt.Sprintf("removing the files of the replaced university %s: %v (prune-files --delete removes orphans)", old, err))
	}
	return warnings, nil
}

func (r *Report) personas(f Fixture) {
	type acc struct{ instr, ta, student []string }
	by := map[string]*acc{}
	for _, p := range f.People {
		by[p.Email] = &acc{}
	}
	add := func(l *[]string, code string) {
		if !slices.Contains(*l, code) {
			*l = append(*l, code)
		}
	}
	for _, c := range f.Courses {
		for _, s := range c.Sections {
			for _, st := range s.Staff {
				if st.Role == "instructor" {
					add(&by[st.Email].instr, c.Code)
				} else {
					add(&by[st.Email].ta, c.Code)
				}
			}
			for _, e := range s.Roster {
				add(&by[e].student, c.Code)
			}
		}
	}
	for _, p := range f.People {
		a := by[p.Email]
		var sees []string
		role := "member"
		switch {
		case p.Invited:
			role = "invited"
		case p.Role == "admin":
			role = "admin"
		case p.Role == "dept_admin":
			role = "department admin"
		case len(a.instr) > 0:
			role = "instructor"
		case len(a.ta) > 0:
			role = "TA"
		case len(a.student) > 0:
			role = "student"
		}
		tail := ""
		if p.Invited {
			sees = append(sees, "invitation pending, never signed in")
			tail = " at first sign-in"
		}
		if p.Role == "admin" {
			sees = append(sees, "admin of "+f.University.Name)
		}
		if p.Role == "dept_admin" {
			sees = append(sees, "department admin of "+strings.Join(p.Departments, ", "))
		}
		if len(a.instr) > 0 {
			sees = append(sees, "instructor of "+strings.Join(a.instr, ", ")+tail)
		}
		if len(a.ta) > 0 {
			sees = append(sees, "TA on "+strings.Join(a.ta, ", ")+tail)
		}
		if len(a.student) > 0 {
			sees = append(sees, "student in "+strings.Join(a.student, ", "))
		}
		if len(sees) == 0 {
			sees = []string{"member, no sections"}
		}
		r.Personas = append(r.Personas, Persona{Role: role, Name: p.DisplayName, Email: p.Email, Sees: strings.Join(sees, "; ")})
	}
}

func (w *writer) seed(db *sql.DB, uni string, f Fixture, o Options) error {
	admin := ""
	for _, p := range f.People {
		w.names[p.Email] = p.DisplayName
		if p.Role == "admin" && !p.Invited {
			admin = p.Email
		}
	}
	// users and memberships
	err := pg.WithPlatformTx(w.ctx, db, func(tx *sql.Tx) error {
		for _, p := range f.People {
			if p.Invited {
				continue
			}
			var id string
			if err := tx.QueryRowContext(w.ctx, `insert into users (email, display_name) values ($1, $2)
				on conflict (email) do update set display_name = excluded.display_name returning id`, p.Email, p.DisplayName).Scan(&id); err != nil {
				return err
			}
			w.users[p.Email] = id
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("users: %w", err)
	}
	err = pg.WithTx(w.ctx, db, uni, func(tx *sql.Tx) error {
		for _, p := range f.People {
			if p.Invited {
				continue
			}
			if _, err := tx.ExecContext(w.ctx, `insert into memberships (university_id, user_id, role, status) values ($1, $2, $3, 'active')`, uni, w.users[p.Email], p.Role); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("memberships: %w", err)
	}

	for _, d := range f.Departments {
		var out idOnly
		if err := w.post("department "+d.Code, admin, http.MethodPost, "/departments", d, &out); err != nil {
			return err
		}
		w.depts[d.Code] = out.ID
	}
	for _, t := range f.Terms {
		var out idOnly
		if err := w.post(fmt.Sprintf("term %q", t.Name), admin, http.MethodPost, "/terms", t, &out); err != nil {
			return err
		}
		w.terms[t.Name] = out.ID
	}

	noMail := func(context.Context, *sql.Tx, ...outbox.Mail) error { return nil }
	err = pg.WithTx(w.ctx, db, uni, func(tx *sql.Tx) error {
		for _, p := range f.People {
			if p.Role == "dept_admin" && !p.Invited {
				for _, c := range p.Departments {
					if _, err := tx.ExecContext(w.ctx, `insert into department_admins (university_id, department_id, user_id) values ($1, $2, $3)`, uni, w.depts[c], w.users[p.Email]); err != nil {
						return fmt.Errorf("department admins: %w", err)
					}
				}
			}
		}
		for _, p := range f.People {
			if !p.Invited {
				continue
			}
			var ids []string
			for _, c := range p.Departments {
				ids = append(ids, w.depts[c])
			}
			inv, err := tenant.CreateInvitation(w.ctx, tx, tenant.NewInvitation{UniversityID: uni, ActorUserID: w.users[admin], Email: p.Email,
				Role: authz.Role(p.Role), DepartmentIDs: ids}, noMail, o.BaseURL)
			if err == nil {
				_, err = tx.ExecContext(w.ctx, `update invitations set display_name = $2 where id = $1`, inv.ID, p.DisplayName)
			}
			if err != nil {
				return fmt.Errorf("invitation %s: %w", p.Email, err)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	for _, c := range f.Courses {
		if err := w.course(c, admin); err != nil {
			return err
		}
	}
	return nil
}

func (w *writer) course(c Course, admin string) error {
	a := cmp.Or(c.As, admin)
	body := struct {
		DepartmentID string      `json:"departmentId"`
		Code         string      `json:"code"`
		Title        string      `json:"title"`
		Credits      json.Number `json:"credits,omitempty"`
	}{w.depts[c.DepartmentCode], c.Code, c.Title, c.Credits}
	var cid idOnly
	if err := w.post("course "+c.Code, a, http.MethodPost, "/courses", body, &cid); err != nil {
		return err
	}
	for _, s := range c.Sections {
		if err := w.section(c, s, cid.ID, a); err != nil {
			return err
		}
	}
	return nil
}

func (w *writer) section(c Course, s Section, courseID, a string) error {
	label := c.Code + " " + s.Number
	meetings := s.Meetings
	if meetings == nil {
		meetings = []Meeting{}
	}
	var sid idOnly
	err := w.post("section "+label, a, http.MethodPost, "/sections", struct {
		CourseID string    `json:"courseId"`
		TermID   string    `json:"termId"`
		Number   string    `json:"number"`
		Meetings []Meeting `json:"meetings"`
	}{courseID, w.terms[s.Term], s.Number, meetings}, &sid)
	if err != nil {
		return err
	}
	base := "/sections/" + sid.ID
	grader := ""
	for _, st := range s.Staff {
		if err := w.post(fmt.Sprintf("staff %s on %s", st.Email, label), a, http.MethodPost, base+"/staff", st, nil); err != nil {
			return err
		}
		if grader == "" && st.Role == "instructor" && w.users[st.Email] != "" {
			grader = st.Email
		}
	}
	if len(s.Roster) > 0 {
		var csvBuf bytes.Buffer
		cw := csv.NewWriter(&csvBuf)
		cw.Write([]string{"email", "displayName"})
		for _, e := range s.Roster {
			cw.Write([]string{e, w.names[e]})
		}
		cw.Flush()
		if err := cw.Error(); err != nil {
			return fmt.Errorf("roster csv of %s: %w", label, err)
		}
		var prev struct {
			Checksum string `json:"checksum"`
		}
		file := part{"file", "roster.csv", "text/csv", csvBuf.Bytes()}
		if err := w.upload("roster preview of "+label, a, base+"/enrollments/import/preview", []part{file}, &prev); err != nil {
			return err
		}
		if err := w.upload("roster import of "+label, a, base+"/enrollments/import", []part{file, {name: "checksum", data: []byte(prev.Checksum)}}, nil); err != nil {
			return err
		}
	}

	if grader != "" {
		if err := w.grade(s, label, base, grader); err != nil {
			return err
		}
	}
	return w.content(s, label, base, cmp.Or(grader, a))
}

func (w *writer) grade(s Section, label, base, i string) error {
	for _, as := range s.Assessments {
		var aid idOnly
		err := w.post(fmt.Sprintf("assessment %q of %s", as.Name, label), i, http.MethodPost, base+"/assessments", struct {
			Name      string      `json:"name"`
			MaxPoints json.Number `json:"maxPoints"`
			Weight    json.Number `json:"weight"`
			Published bool        `json:"published"`
		}{as.Name, as.MaxPoints, as.Weight, as.Published}, &aid)
		if err != nil {
			return err
		}
		if len(as.Scores) == 0 {
			continue
		}
		type input struct {
			UserID string      `json:"userId"`
			Points json.Number `json:"points"`
		}
		var in []input
		for _, sc := range as.Scores {
			in = append(in, input{w.users[sc.Student], sc.Points})
		}
		err = w.post(fmt.Sprintf("scores of %q in %s", as.Name, label), i, http.MethodPut, base+"/assessments/"+aid.ID+"/scores", struct {
			Scores []input `json:"scores"`
		}{in}, nil)
		if err != nil {
			return err
		}
	}
	if s.Finalize == nil {
		return nil
	}
	type override struct {
		UserID string `json:"userId"`
		Status string `json:"status,omitempty"`
		Letter string `json:"letter,omitempty"`
		Reason string `json:"reason"`
	}
	ov := []override{}
	for _, o := range s.Finalize.Overrides {
		ov = append(ov, override{w.users[o.Student], o.Status, o.Letter, o.Reason})
	}
	return w.post("final grades of "+label, i, http.MethodPost, base+"/final-grades", struct {
		Overrides []override `json:"overrides"`
	}{ov}, nil)
}

func (w *writer) content(s Section, label, base, cActor string) error {
	if s.Description != "" {
		if err := w.post("description of "+label, cActor, http.MethodPut, base+"/description", map[string]string{"description": s.Description}, nil); err != nil {
			return err
		}
	}
	for _, an := range s.Announcements {
		err := w.post(fmt.Sprintf("announcement %q in %s", an.Title, label), cmp.Or(an.As, cActor), http.MethodPost, base+"/announcements",
			struct {
				Title string `json:"title"`
				Body  string `json:"body"`
				Email bool   `json:"email"`
			}{an.Title, an.Body, false}, nil)
		if err != nil {
			return err
		}
	}
	for _, wk := range s.Weeks {
		var wid idOnly
		err := w.post(fmt.Sprintf("week %q in %s", wk.Title, label), cActor, http.MethodPost, base+"/weeks",
			struct {
				Title string `json:"title"`
				Notes string `json:"notes"`
			}{wk.Title, wk.Notes}, &wid)
		if err != nil {
			return err
		}
		for _, m := range wk.Materials {
			data, err := files.ReadFile("files/" + m.File)
			if err != nil {
				return err
			}
			var parts []part
			if m.Title != "" {
				parts = append(parts, part{name: "title", data: []byte(m.Title)})
			}
			parts = append(parts, part{"file", m.File, "application/octet-stream", data})
			if err := w.upload(fmt.Sprintf("material %s in %s", m.File, label), cActor, base+"/weeks/"+wid.ID+"/materials", parts, nil); err != nil {
				return err
			}
		}
	}
	return nil
}
