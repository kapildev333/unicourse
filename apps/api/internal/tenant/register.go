package tenant

import "github.com/kapildev333/unicourse/apps/api/internal/httpx"

const base = "/api/v1/universities/{slug}"

// Register mounts every tenant-scoped route built so far; each goes through Wrap.
func Register(r *httpx.Router, d Deps) {
	r.Handle("GET", base+"/departments", Wrap(d, listDepartments))
	r.Handle("POST", base+"/departments", Wrap(d, createDepartment))
	r.Handle("PATCH", base+"/departments/{id}", Wrap(d, updateDepartment))
	r.Handle("DELETE", base+"/departments/{id}", Wrap(d, deleteDepartment))
	r.Handle("GET", base+"/terms", Wrap(d, listTerms))
	r.Handle("POST", base+"/terms", Wrap(d, createTerm))
	r.Handle("PATCH", base+"/terms/{id}", Wrap(d, updateTerm))
	r.Handle("DELETE", base+"/terms/{id}", Wrap(d, deleteTerm))
}
