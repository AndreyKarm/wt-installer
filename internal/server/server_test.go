package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsAllowedOrigin(t *testing.T) {
	for origin, want := range map[string]bool{
		"":                            true,
		"https://live.warthunder.com": true,
		"moz-extension://abc-123":     true,
		"https://evil.example":        false,
		"https://live.warthunder.com.evil.example": false,
	} {
		if got := isAllowedOrigin(origin); got != want {
			t.Errorf("%q: got %v, want %v", origin, got, want)
		}
	}
}

func post(t *testing.T, h http.HandlerFunc, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodPost, target, strings.NewReader(body)))
	return rec
}

// These cases fail before touching game folders or the installed store.
func TestHandlersRejectBadRequests(t *testing.T) {
	cases := []struct {
		name       string
		h          http.HandlerFunc
		target     string
		body       string
		wantStatus int
		wantBody   string
	}{
		{"uninstall no body", uninstallHandler, "/uninstall", "", 400, "Invalid JSON"},
		{"uninstall no id", uninstallHandler, "/uninstall", `{}`, 400, "Invalid JSON"},
		{"download bad type", downloadHandler, "/download", `{"type":"model","postId":1}`, 400, "Invalid type"},
		{"download non-WT host", downloadHandler, "/download",
			`{"type":"sight","postId":1,"fileUrl":"https://example.com/x.zip"}`, 500, "only allowed from War Thunder"},
		{"openfolder bad target", openFolderHandler, "/openfolder?target=nope", "", 400, "invalid target"},
	}
	for _, c := range cases {
		rec := post(t, c.h, c.target, c.body)
		if rec.Code != c.wantStatus || !strings.Contains(rec.Body.String(), c.wantBody) {
			t.Errorf("%s: got %d %q, want %d containing %q", c.name, rec.Code, rec.Body.String(), c.wantStatus, c.wantBody)
		}
	}

	rec := httptest.NewRecorder()
	uninstallHandler(rec, httptest.NewRequest(http.MethodGet, "/uninstall", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /uninstall: got %d", rec.Code)
	}
}
