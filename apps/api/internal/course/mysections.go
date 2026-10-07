package course

import (
	"database/sql"
	"net/http"

	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

type MySection struct {
	Section
	MyRole string `json:"myRole"`
}

type TermSections struct {
	Term     tenant.Term `json:"term"`
	Sections []MySection `json:"sections"`
}

type MySections struct {
	Terms []TermSections `json:"terms"`
}

// getMySections lists the caller's sections (staff role wins over enrollment), newest term first.
// Unpaginated: bounded by one user's own rows. RLS keeps other universities out.
func getMySections(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	rows, err := tx.QueryContext(r.Context(), `with mine as (
		select section_id, role as my_role from section_staff where user_id = $1
		union all
		select e.section_id, 'student' from enrollments e
		where e.user_id = $1 and not exists (select 1 from section_staff ss where ss.section_id = e.section_id and ss.user_id = $1)
	) select `+sectionCols+`, mine.my_role`+sectionFrom+` join mine on mine.section_id = s.id
	order by t.starts_on desc, t.id, c.code, s.number`, s.UserID)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	out := MySections{Terms: []TermSections{}}
	for rows.Next() {
		var role string
		sec, err := scanSection(rows, &role)
		if err != nil {
			return 0, nil, err
		}
		if plainMember(s) && role == "student" {
			sec.PendingStaff = []PendingStaff{}
		}
		if n := len(out.Terms); n == 0 || out.Terms[n-1].Term.ID != sec.Term.ID {
			out.Terms = append(out.Terms, TermSections{Term: sec.Term, Sections: []MySection{}})
		}
		g := &out.Terms[len(out.Terms)-1]
		g.Sections = append(g.Sections, MySection{Section: sec, MyRole: role})
	}
	return http.StatusOK, out, rows.Err()
}
