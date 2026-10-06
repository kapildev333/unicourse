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
	r.Handle("GET", base+"/memberships", Wrap(d, listMemberships))
	r.Handle("PATCH", base+"/memberships/{id}", Wrap(d, updateMembership))
	r.Handle("DELETE", base+"/memberships/{id}", Wrap(d, deleteMembership))
	r.Handle("GET", base+"/invitations", Wrap(d, listInvitations))
	r.Handle("POST", base+"/invitations", Wrap(d, createInvitation(d)))
	r.Handle("DELETE", base+"/invitations/{id}", Wrap(d, deleteInvitation))
	r.Handle("GET", base+"/audit-log", Wrap(d, listAuditLog))
}
