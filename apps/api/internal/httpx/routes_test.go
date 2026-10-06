package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestRouter(t *testing.T) {
	r := NewRouter()
	r.Handle("GET", "/api/v1/things/{id}", func(w http.ResponseWriter, req *http.Request) error {
		WriteJSON(w, 200, map[string]string{"id": req.PathValue("id")})
		return nil
	})
	r.Handle("POST", "/api/v1/things", func(w http.ResponseWriter, req *http.Request) error { return NotFound("thing") })

	do := func(method, path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(method, path, nil))
		return w
	}

	if w := do("GET", "/api/v1/things/42"); w.Code != 200 || w.Body.String() != "{\"id\":\"42\"}\n" && w.Body.String() != "{\"id\":\"42\"}" {
		t.Fatalf("path value: %d %q", w.Code, w.Body.String())
	}
	if w := do("POST", "/api/v1/things"); w.Code != 404 {
		t.Fatalf("handler error: %d", w.Code)
	}
	for _, c := range [][2]string{{"GET", "/api/v1/nope"}, {"DELETE", "/api/v1/things/42"}} {
		w := do(c[0], c[1])
		var env struct {
			Error struct{ Code, Message string }
		}
		if w.Code != 404 || json.Unmarshal(w.Body.Bytes(), &env) != nil || env.Error.Code != "not_found" {
			t.Fatalf("%v: %d %q", c, w.Code, w.Body.String())
		}
	}
	want := []Route{{"GET", "/api/v1/things/{id}"}, {"POST", "/api/v1/things"}}
	if got := r.Routes(); !reflect.DeepEqual(got, want) {
		t.Fatalf("routes %v", got)
	}
}

func TestRouterDuplicatePanics(t *testing.T) {
	r := NewRouter()
	h := func(http.ResponseWriter, *http.Request) error { return nil }
	r.Handle("GET", "/x", h)
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	r.Handle("GET", "/x", h)
}
