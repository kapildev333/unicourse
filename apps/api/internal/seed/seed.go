// Package seed loads the demo university: Load validates the embedded fixture, Run writes it through
// the API's own handlers (plus the few rows no handler can write).
package seed

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kapildev333/unicourse/apps/api/internal/auth"
)

//go:embed demo.json
var Demo []byte

//go:embed files
var files embed.FS // read as files.ReadFile("files/" + name)

type Fixture struct {
	University  University   `json:"university"`
	Departments []Department `json:"departments"`
	Terms       []Term       `json:"terms"`
	People      []Person     `json:"people"`
	Courses     []Course     `json:"courses"`

	files fs.FS // where materials' files are read; set by Load/LoadFile, nil reads the embedded files
}
type University struct {
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Timezone string `json:"timezone"`
	Locale   string `json:"locale"`
}
type Department struct {
	Name string `json:"name"`
	Code string `json:"code"`
}
type Term struct {
	Name     string `json:"name"`
	StartsOn string `json:"startsOn"`
	EndsOn   string `json:"endsOn"`
}
type Person struct {
	Email       string   `json:"email"`
	DisplayName string   `json:"displayName"`
	Role        string   `json:"role"`
	Departments []string `json:"departments,omitempty"`
	Invited     bool     `json:"invited,omitempty"`
}
type Course struct {
	As             string      `json:"as,omitempty"`
	DepartmentCode string      `json:"departmentCode"`
	Code           string      `json:"code"`
	Title          string      `json:"title"`
	Credits        json.Number `json:"credits,omitempty"`
	Sections       []Section   `json:"sections"`
}
type Section struct {
	Term          string         `json:"term"`
	Number        string         `json:"number"`
	Meetings      []Meeting      `json:"meetings,omitempty"`
	Staff         []Staff        `json:"staff,omitempty"`
	Roster        []string       `json:"roster,omitempty"`
	Description   string         `json:"description,omitempty"`
	Assessments   []Assessment   `json:"assessments,omitempty"`
	Finalize      *Finalize      `json:"finalize,omitempty"`
	Announcements []Announcement `json:"announcements,omitempty"`
	Weeks         []Week         `json:"weeks,omitempty"`
}
type Meeting struct {
	Weekday  int    `json:"weekday"`
	StartsAt string `json:"startsAt"`
	EndsAt   string `json:"endsAt"`
	Location string `json:"location"`
}
type Staff struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}
type Assessment struct {
	Name      string      `json:"name"`
	MaxPoints json.Number `json:"maxPoints"`
	Weight    json.Number `json:"weight"`
	Published bool        `json:"published"`
	Scores    []Score     `json:"scores,omitempty"`
}
type Score struct {
	Student string      `json:"student"`
	Points  json.Number `json:"points"`
}
type Finalize struct {
	Overrides []Override `json:"overrides"`
}
type Override struct {
	Student string `json:"student"`
	Status  string `json:"status,omitempty"`
	Letter  string `json:"letter,omitempty"`
	Reason  string `json:"reason"`
}
type Announcement struct {
	As    string `json:"as,omitempty"`
	Title string `json:"title"`
	Body  string `json:"body"`
}
type Week struct {
	Title     string     `json:"title"`
	Notes     string     `json:"notes"`
	Materials []Material `json:"materials,omitempty"`
}
type Material struct {
	File  string `json:"file"`
	Title string `json:"title"`
}

// Load substitutes the placeholders (slug "" = the fixture's slug; year is <year>) and validates the
// shape and references. Value rules (lengths, formats, weights) are the API's, reported at write time.
func Load(raw []byte, slug string, year int) (Fixture, error) {
	sub, err := fs.Sub(files, "files")
	if err != nil {
		return Fixture{}, err
	}
	return load(raw, "demo.json", sub, slug, year)
}

// LoadFile is Load for a fixture file on disk; its materials' files resolve relative to the file's directory.
func LoadFile(path, slug string, year int) (Fixture, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Fixture{}, err
	}
	return load(raw, filepath.Base(path), os.DirFS(filepath.Dir(path)), slug, year)
}

func load(raw []byte, name string, files fs.FS, slug string, year int) (Fixture, error) {
	var probe struct{ University struct{ Slug string } }
	if err := json.Unmarshal(raw, &probe); err != nil {
		return Fixture{}, fmt.Errorf("%s: %w", name, err)
	}
	if slug == "" {
		slug = probe.University.Slug
	}
	sub := strings.NewReplacer("<year+1>", strconv.Itoa(year+1), "<year>", strconv.Itoa(year), "<domain>", slug+".edu").Replace(string(raw))
	dec := json.NewDecoder(strings.NewReader(sub))
	dec.DisallowUnknownFields()
	var f Fixture
	if err := dec.Decode(&f); err != nil {
		return Fixture{}, fmt.Errorf("%s: %w", name, err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return Fixture{}, errors.New(name + ": unexpected data after the top-level object")
	}
	f.University.Slug, f.files = slug, files
	if err := f.validate(); err != nil {
		return Fixture{}, fmt.Errorf("%s: %w", name, err)
	}
	return f, nil
}

func (f *Fixture) validate() error {
	for _, k := range []struct{ path, v string }{{"university.name", f.University.Name}, {"university.slug", f.University.Slug},
		{"university.timezone", f.University.Timezone}, {"university.locale", f.University.Locale}} {
		if k.v == "" {
			return fmt.Errorf("%s: is required", k.path)
		}
	}
	depts := map[string]bool{}
	for i, d := range f.Departments {
		if depts[d.Code] {
			return fmt.Errorf("departments[%d].code: duplicate %q", i, d.Code)
		}
		depts[d.Code] = true
	}
	terms := map[string]bool{}
	for i, t := range f.Terms {
		if terms[t.Name] {
			return fmt.Errorf("terms[%d].name: duplicate %q", i, t.Name)
		}
		terms[t.Name] = true
	}
	invited := map[string]bool{} // normalised email -> invited
	admins := 0
	for i := range f.People {
		p := &f.People[i]
		e, err := auth.NormalizeEmail(p.Email)
		if err != nil {
			return fmt.Errorf("people[%d].email: %q is not a valid email", i, p.Email)
		}
		if _, dup := invited[e]; dup {
			return fmt.Errorf("people[%d].email: duplicate %q", i, e)
		}
		p.Email = e
		invited[e] = p.Invited
		if p.DisplayName == "" {
			return fmt.Errorf("people[%d].displayName: is required", i)
		}
		switch p.Role {
		case "admin":
			if !p.Invited {
				admins++
			}
		case "dept_admin", "member":
		default:
			return fmt.Errorf("people[%d].role: %q is not admin, dept_admin or member", i, p.Role)
		}
		if p.Role == "dept_admin" && len(p.Departments) == 0 {
			return fmt.Errorf("people[%d].departments: required for dept_admin", i)
		}
		if p.Role != "dept_admin" && len(p.Departments) > 0 {
			return fmt.Errorf("people[%d].departments: only for dept_admin", i)
		}
		for _, c := range p.Departments {
			if !depts[c] {
				return fmt.Errorf("people[%d].departments: unknown department %q", i, c)
			}
		}
	}
	if admins != 1 {
		return fmt.Errorf("people: want exactly one admin who is not invited, have %d", admins)
	}
	// ref resolves a referenced email; mustBeActive refuses an invited person.
	ref := func(path, email string, mustBeActive bool) (string, error) {
		e, err := auth.NormalizeEmail(email)
		if err != nil {
			return "", fmt.Errorf("%s: %q is not a valid email", path, email)
		}
		inv, ok := invited[e]
		if !ok {
			return "", fmt.Errorf("%s: %q is not in people", path, email)
		}
		if inv && mustBeActive {
			return "", fmt.Errorf("%s: %q is invited and cannot act or be enrolled", path, email)
		}
		return e, nil
	}
	courses, sections := map[string]bool{}, map[string]bool{}
	for ci := range f.Courses {
		c := &f.Courses[ci]
		cp := fmt.Sprintf("courses[%d]", ci)
		if c.As != "" {
			var err error
			if c.As, err = ref(cp+".as", c.As, true); err != nil {
				return err
			}
		}
		if !depts[c.DepartmentCode] {
			return fmt.Errorf("%s.departmentCode: unknown department %q", cp, c.DepartmentCode)
		}
		if courses[c.Code] {
			return fmt.Errorf("%s.code: duplicate %q", cp, c.Code)
		}
		courses[c.Code] = true
		for si := range c.Sections {
			s := &c.Sections[si]
			sp := fmt.Sprintf("%s.sections[%d]", cp, si)
			if !terms[s.Term] {
				return fmt.Errorf("%s.term: unknown term %q", sp, s.Term)
			}
			if k := c.Code + "/" + s.Term + "/" + s.Number; sections[k] {
				return fmt.Errorf("%s: duplicate section %s", sp, k)
			} else {
				sections[k] = true
			}
			instructor := false
			for i := range s.Staff {
				e, err := ref(fmt.Sprintf("%s.staff[%d].email", sp, i), s.Staff[i].Email, false)
				if err != nil {
					return err
				}
				s.Staff[i].Email = e
				if s.Staff[i].Role == "instructor" && !invited[e] {
					instructor = true
				}
			}
			inRoster := map[string]bool{}
			for i := range s.Roster {
				e, err := ref(fmt.Sprintf("%s.roster[%d]", sp, i), s.Roster[i], true)
				if err != nil {
					return err
				}
				if inRoster[e] {
					return fmt.Errorf("%s.roster[%d]: duplicate %q", sp, i, e)
				}
				s.Roster[i], inRoster[e] = e, true
			}
			if (len(s.Assessments) > 0 || s.Finalize != nil) && !instructor {
				return fmt.Errorf("%s: assessments or finalize need an active instructor in staff", sp)
			}
			for ai := range s.Assessments {
				seen := map[string]bool{}
				for i := range s.Assessments[ai].Scores {
					sc := &s.Assessments[ai].Scores[i]
					path := fmt.Sprintf("%s.assessments[%d].scores[%d].student", sp, ai, i)
					e, err := ref(path, sc.Student, true)
					if err != nil {
						return err
					}
					if !inRoster[e] || seen[e] {
						return fmt.Errorf("%s: %q is not in this section's roster or is scored twice", path, e)
					}
					sc.Student, seen[e] = e, true
				}
			}
			if s.Finalize != nil {
				seen := map[string]bool{}
				for i := range s.Finalize.Overrides {
					o := &s.Finalize.Overrides[i]
					path := fmt.Sprintf("%s.finalize.overrides[%d].student", sp, i)
					e, err := ref(path, o.Student, true)
					if err != nil {
						return err
					}
					if !inRoster[e] || seen[e] {
						return fmt.Errorf("%s: %q is not in this section's roster or has two overrides", path, e)
					}
					o.Student, seen[e] = e, true
				}
			}
			for i := range s.Announcements {
				if a := &s.Announcements[i]; a.As != "" {
					var err error
					if a.As, err = ref(fmt.Sprintf("%s.announcements[%d].as", sp, i), a.As, true); err != nil {
						return err
					}
				}
			}
			for wi, w := range s.Weeks {
				for mi, m := range w.Materials {
					path := fmt.Sprintf("%s.weeks[%d].materials[%d].file", sp, wi, mi)
					if strings.Contains(m.File, "/") {
						return fmt.Errorf("%s: %q must be a bare file name", path, m.File)
					}
					if _, err := fs.Stat(f.files, m.File); err != nil {
						return fmt.Errorf("%s: no embedded file %q", path, m.File)
					}
				}
			}
		}
	}
	return nil
}
