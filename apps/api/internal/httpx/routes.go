package httpx

import "net/http"

// Route is one registered method and pattern, e.g. {"GET", "/api/v1/me"}.
type Route struct{ Method, Pattern string }

// Router wraps http.ServeMux, records what is registered (for the contract
// test) and answers unmatched requests with the JSON error envelope.
type Router struct {
	mux    *http.ServeMux
	routes []Route
}

func NewRouter() *Router { return &Router{mux: http.NewServeMux()} }

// Handle registers h for method and pattern; a duplicate panics (ServeMux).
func (rt *Router) Handle(method, pattern string, h Handler) {
	rt.mux.HandleFunc(method+" "+pattern, H(h))
	rt.routes = append(rt.routes, Route{method, pattern})
}

// Routes returns the registered routes in registration order.
func (rt *Router) Routes() []Route { return append([]Route(nil), rt.routes...) }

// ServeHTTP answers 404 not_found for both an unknown path and a wrong method.
func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, pattern := rt.mux.Handler(r)
	if pattern == "" {
		WriteError(w, r, NotFound("route"))
		return
	}
	setRoute(r.Context(), pattern)
	rt.mux.ServeHTTP(w, r)
}
