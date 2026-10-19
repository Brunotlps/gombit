package auth

import (
	"errors"
	"fmt"
	"strings"
)

var (
	errPasswordRequired   = errors.New("password is required")
	errPasswordTooLong    = errors.New("password must be at most 72 bytes")
	errInvalidCredentials = errors.New("invalid email or password")
	errEmailTaken         = errors.New("email already registered")
	errUserNotFound       = errors.New("user not found")
	errRefreshReuse       = errors.New("refresh token reuse detected")
	// errRotationRaced is a rotation that lost to another change of its
	// token. Either it lost the compare-and-swap, almost always to a
	// rotation of the same token that another instance ran (one instance
	// joins its concurrent callers of a token to one rotation), sometimes to
	// a session revocation, or it found the token expired and already given
	// a successor. It is an invalid refresh token, a 401; refreshCookie
	// leaves the cookies alone for it, because a winning rotation's response
	// sets the session's new ones in the same browser.
	errRotationRaced = fmt.Errorf("%w: rotated concurrently", errInvalidRefreshToken)
)

// ErrEmailTaken is the exported form of errEmailTaken. Register and
// CreateSuperuser return it (wrapped) so callers such as gombit
// createsuperuser can detect duplicates with errors.Is without a second
// uniqueness check.
var ErrEmailTaken = errEmailTaken

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
