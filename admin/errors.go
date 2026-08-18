package admin

import (
	"encoding/json"
	"fmt"

	"github.com/LAA-Software-Engineering/gombit/contract"
	"github.com/danielgtaylor/huma/v2"
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

func writeEnvelope(ctx huma.Context, env *contract.ErrorEnvelope) {
	if env == nil {
		return
	}
	ctx.SetHeader("Content-Type", "application/json")
	ctx.SetStatus(env.GetStatus())
	_ = json.NewEncoder(ctx.BodyWriter()).Encode(env)
}
