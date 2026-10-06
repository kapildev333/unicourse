package course

import (
	"database/sql"
	"encoding/json"
	"testing"
)

func TestMySections(t *testing.T) {
	e := newEnv(t)
	me := e.UserID["other@a.edu"]
	s2 := e.newSection(e.TermA2, "001")
	s3 := e.newSection(e.TermA2, "002")
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		for _, q := range [][]any{
			{`insert into section_staff (university_id, section_id, user_id, role) values ($1,$2,$3,'instructor')`, e.SectionA},
			{`insert into section_staff (university_id, section_id, user_id, role) values ($1,$2,$3,'ta')`, s2},
			{`insert into enrollments (university_id, section_id, user_id) values ($1,$2,$3)`, s3},
			{`insert into enrollments (university_id, section_id, user_id) values ($1,$2,$3)`, s2}, // staff wins
		} {
			if _, err := tx.Exec(q[0].(string), e.Fx.UniversityA, q[1], me); err != nil {
				return err
			}
		}
		return nil
	})
	// the same user has a section in university B: never visible under A
	e.tx(e.Fx.UniversityB, func(tx *sql.Tx) error {
		var sec string
		if _, err := tx.Exec(`insert into memberships (university_id, user_id, role, status) values ($1,$2,'member','active')`, e.Fx.UniversityB, me); err != nil {
			return err
		}
		if err := tx.QueryRow(`insert into sections (university_id, course_id, term_id, number) values ($1,$2,$3,'9') returning id`, e.Fx.UniversityB, e.CourseB, e.TermB).Scan(&sec); err != nil {
			return err
		}
		_, err := tx.Exec(`insert into section_staff (university_id, section_id, user_id, role) values ($1,$2,$3,'instructor')`, e.Fx.UniversityB, sec, me)
		return err
	})
	st, raw := e.call(e.Other, "GET", "/me/sections", nil)
	var got MySections
	if st != 200 || json.Unmarshal(raw, &got) != nil || len(got.Terms) != 2 {
		t.Fatalf("my sections: %d %s", st, raw)
	}
	if got.Terms[0].Term.ID != e.TermA2 || got.Terms[1].Term.ID != e.Fx.TermA {
		t.Errorf("terms not newest first: %s", raw)
	}
	roles := map[string]string{}
	for _, g := range got.Terms {
		for _, s := range g.Sections {
			roles[s.ID] = s.MyRole
			if s.Course.Code != "CS101" || s.Staff == nil || s.PendingStaff == nil {
				t.Errorf("section shape: %+v", s)
			}
		}
	}
	if len(roles) != 3 || roles[e.SectionA] != "instructor" || roles[s2] != "ta" || roles[s3] != "student" {
		t.Errorf("roles: %v", roles)
	}
	if m := jmap(t, raw)["terms"].([]any)[0].(map[string]any)["sections"].([]any)[0].(map[string]any); m["myRole"] == nil || m["courseId"] == nil {
		t.Errorf("myRole not flattened beside section fields: %v", m)
	}
}

func TestMySectionsEmpty(t *testing.T) {
	e := newEnv(t)
	st, raw := e.call(e.DaOut, "GET", "/me/sections", nil)
	if st != 200 || string(raw) != "{\"terms\":[]}\n" {
		t.Errorf("empty: %d %q", st, raw)
	}
	if st, _ := e.call(nil, "GET", "/me/sections", nil); st != 401 {
		t.Errorf("anonymous: %d", st)
	}
}
