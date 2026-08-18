package admin

import (
	"fmt"
)

func errMissingSlug() error {
	return fmt.Errorf("admin: missing slug")
}

func errDuplicateSlug(slug string) error {
	return fmt.Errorf("admin: duplicate slug %q", slug)
}

func errInvalidSlug(slug string) error {
	return fmt.Errorf("admin: invalid slug %q", slug)
}
