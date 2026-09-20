package auth

import (
	"net/http/httptest"
	"testing"
)

func TestSetSessionCookieUsesIdleTimeout(t *testing.T) {
	recorder := httptest.NewRecorder()
	SetSessionCookie(recorder, "token")

	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
	if cookies[0].MaxAge != sessionCookieMaxAge {
		t.Fatalf("MaxAge = %d, want %d", cookies[0].MaxAge, sessionCookieMaxAge)
	}
}
