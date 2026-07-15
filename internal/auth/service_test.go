package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSessionCookieRoundTrip(t *testing.T) {
	token := "test-session-token"
	expiresAt := time.Now().Add(2 * time.Hour).UTC()

	cookie := buildSessionCookie(token, expiresAt, false)
	if cookie.Name != sessionCookieName {
		t.Fatalf("expected cookie name %q, got %q", sessionCookieName, cookie.Name)
	}
	if cookie.Path != "/" {
		t.Fatalf("expected cookie path %q, got %q", "/", cookie.Path)
	}
	if cookie.Value != token {
		t.Fatalf("expected cookie value %q, got %q", token, cookie.Value)
	}

	req := httptest.NewRequest(http.MethodGet, "http://localhost/api/auth/me", nil)
	req.AddCookie(cookie)

	readToken, ok := readSessionCookie(req)
	if !ok {
		t.Fatal("expected session cookie to be readable")
	}
	if readToken != token {
		t.Fatalf("expected token %q, got %q", token, readToken)
	}
}

func TestClearSessionCookie(t *testing.T) {
	cookie := clearSessionCookie(false)
	if cookie.Name != sessionCookieName {
		t.Fatalf("expected cookie name %q, got %q", sessionCookieName, cookie.Name)
	}
	if cookie.Path != "/" {
		t.Fatalf("expected cookie path %q, got %q", "/", cookie.Path)
	}
	if cookie.MaxAge != -1 {
		t.Fatalf("expected cookie max age -1, got %d", cookie.MaxAge)
	}
	if cookie.Value != "" {
		t.Fatalf("expected cleared cookie value to be empty, got %q", cookie.Value)
	}
}

func TestSessionTokenFingerprint(t *testing.T) {
	fingerprint := SessionTokenFingerprint("test-session-token")
	if len(fingerprint) != 12 {
		t.Fatalf("expected 12 character fingerprint, got %q", fingerprint)
	}
}
