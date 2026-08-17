package config

import (
	"net/url"
	"regexp"
	"strings"
)

// RedactedSecret is the placeholder used in place of passwords and DSN userinfo.
const RedactedSecret = "*****"

var querySecretPattern = regexp.MustCompile(`(?i)((?:password|pwd|pass)=)([^&]*)`)

// Redacted returns a copy of c with secret-bearing fields replaced so it is
// safe to print (DSN userinfo/passwords and Redis password).
func (c Config) Redacted() Config {
	out := c
	out.Database.DSN = RedactDSN(c.Database.DSN)
	if strings.TrimSpace(c.Cache.Redis.Password) != "" {
		out.Cache.Redis.Password = RedactedSecret
	}
	return out
}

// RedactDSN returns dsn with userinfo passwords and password-like query
// parameters replaced. SQLite file paths without userinfo are unchanged.
func RedactDSN(dsn string) string {
	trimmed := strings.TrimSpace(dsn)
	if trimmed == "" {
		return dsn
	}

	if strings.Contains(trimmed, "://") {
		u, err := url.Parse(trimmed)
		if err == nil && u.Scheme != "" {
			if u.User != nil {
				if _, hasPassword := u.User.Password(); hasPassword {
					u.User = url.UserPassword(u.User.Username(), RedactedSecret)
				}
			}
			return redactQuerySecrets(u.String())
		}
	}

	if !strings.HasPrefix(trimmed, "file:") {
		if at := strings.Index(trimmed, "@"); at > 0 {
			userinfo := trimmed[:at]
			if colon := strings.Index(userinfo, ":"); colon >= 0 {
				return redactQuerySecrets(userinfo[:colon+1] + RedactedSecret + trimmed[at:])
			}
		}
	}

	return redactQuerySecrets(trimmed)
}

// SanitizeError returns err.Error() with known secrets from cfg removed.
func SanitizeError(err error, cfg Config) string {
	if err == nil {
		return ""
	}
	return SanitizeSecretText(err.Error(), cfg)
}

// SanitizeSecretText replaces known secrets from cfg inside text.
func SanitizeSecretText(text string, cfg Config) string {
	if text == "" {
		return text
	}
	if dsn := strings.TrimSpace(cfg.Database.DSN); dsn != "" {
		text = strings.ReplaceAll(text, dsn, RedactDSN(dsn))
		if u, err := url.Parse(dsn); err == nil && u.User != nil {
			if password, ok := u.User.Password(); ok && password != "" {
				text = strings.ReplaceAll(text, password, RedactedSecret)
			}
		}
		if !strings.HasPrefix(dsn, "file:") {
			if at := strings.Index(dsn, "@"); at > 0 {
				userinfo := dsn[:at]
				if colon := strings.Index(userinfo, ":"); colon >= 0 {
					password := userinfo[colon+1:]
					if password != "" {
						text = strings.ReplaceAll(text, password, RedactedSecret)
					}
				}
			}
		}
	}
	if password := cfg.Cache.Redis.Password; password != "" {
		text = strings.ReplaceAll(text, password, RedactedSecret)
	}
	return text
}

func redactQuerySecrets(value string) string {
	return querySecretPattern.ReplaceAllString(value, "${1}"+RedactedSecret)
}
