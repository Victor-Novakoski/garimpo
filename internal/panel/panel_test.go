package panel

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func get(t *testing.T, h http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func TestServesTheBuild(t *testing.T) {
	h := handler(fstest.MapFS{
		"index.html":      {Data: []byte("<!doctype html><title>Garimpo</title>")},
		"assets/app-1.js": {Data: []byte("console.log(1)")},
	})
	rec := get(t, h, "GET", "/")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "Garimpo") || rec.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("index = %d %q %v", rec.Code, rec.Body.String(), rec.Header())
	}
	for _, header := range []string{"Content-Security-Policy", "X-Content-Type-Options", "X-Frame-Options", "Referrer-Policy"} {
		if rec.Header().Get(header) == "" {
			t.Errorf("index without %s", header)
		}
	}
	rec = get(t, h, "GET", "/assets/app-1.js")
	if rec.Code != 200 || !strings.Contains(rec.Header().Get("Cache-Control"), "immutable") {
		t.Errorf("asset = %d %v", rec.Code, rec.Header())
	}
	if rec = get(t, h, "GET", "/nada.js"); rec.Code != 404 {
		t.Errorf("missing file = %d, want 404", rec.Code)
	}
	if rec = get(t, h, "POST", "/"); rec.Code != 405 {
		t.Errorf("post = %d, want 405", rec.Code)
	}
}

func TestWithoutTheBuildItSaysHowToMakeIt(t *testing.T) {
	rec := get(t, handler(fstest.MapFS{".gitkeep": {}}), "GET", "/")
	if rec.Code != 503 || !strings.Contains(rec.Body.String(), "npm --prefix web run build") {
		t.Errorf("not built = %d %q", rec.Code, rec.Body.String())
	}
}

func TestTheEmbeddedDirectoryOpens(t *testing.T) {
	// With or without the build, the handler must come up.
	if rec := get(t, Handler(), "GET", "/"); rec.Code != 200 && rec.Code != 503 {
		t.Errorf("embedded panel = %d", rec.Code)
	}
}
