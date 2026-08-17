package auth

import (
	"net/http"
	"time"

	"github.com/LAA-Software-Engineering/gombit/config"
)

// Cookie and CSRF header names for cookie-mode session auth (M5-3). See
// docs/auth-cookie.md for the threat model these pair with.
const (
	// AccessCookieName holds the short-lived access JWT in cookie mode.
	AccessCookieName = "gombit_access"
	// RefreshCookieName holds the rotating refresh token in cookie mode.
	RefreshCookieName = "gombit_refresh"
	// CSRFCookieName holds the signed double-submit CSRF token.
	CSRFCookieName = "gombit_csrf"
	// CSRFHeaderName is the header clients must echo the CSRF cookie value
	// into for state-changing (POST/PUT/PATCH/DELETE) requests.
	CSRFHeaderName = "X-CSRF-Token"

	// csrfCookieMaxAge bounds the CSRF cookie lifetime independently of the
	// session cookies: it is reissued on any safe request once missing.
	csrfCookieMaxAge = 24 * time.Hour
)

func cookieSameSite(mode config.CookieSameSite) http.SameSite {
	if mode == config.CookieSameSiteStrict {
		return http.SameSiteStrictMode
	}
	return http.SameSiteLaxMode
}

// newSessionCookie builds an HttpOnly access/refresh session cookie whose
// Secure and SameSite attributes come from cfg (Appendix C: production must
// set CookieSecure; config.Validate enforces this).
func newSessionCookie(name, value string, ttl time.Duration, cfg config.AuthConfig) http.Cookie {
	return http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   int(ttl.Seconds()),
		Secure:   cfg.CookieSecure,
		HttpOnly: true,
		SameSite: cookieSameSite(cfg.EffectiveCookieSameSite()),
	}
}

// expiredSessionCookie clears a previously set session cookie (logout).
func expiredSessionCookie(name string, cfg config.AuthConfig) http.Cookie {
	return http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Secure:   cfg.CookieSecure,
		HttpOnly: true,
		SameSite: cookieSameSite(cfg.EffectiveCookieSameSite()),
	}
}

// csrfCookie builds the double-submit CSRF cookie. Unlike the session
// cookies it is not HttpOnly: the SPA reads it (or the mirrored
// csrf_token response body) and echoes it back as CSRFHeaderName.
func csrfCookie(value string, cfg config.AuthConfig) http.Cookie {
	return http.Cookie{
		Name:     CSRFCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   int(csrfCookieMaxAge.Seconds()),
		Secure:   cfg.CookieSecure,
		HttpOnly: false,
		SameSite: cookieSameSite(cfg.EffectiveCookieSameSite()),
	}
}
