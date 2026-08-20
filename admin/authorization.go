package admin

import (
	"context"

	"github.com/gombit-dev/gombit/auth"
	"github.com/gombit-dev/gombit/contract"
)

func (h *handlers) permissionGrants(ctx context.Context, keys ...string) (map[string]bool, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, contract.WithContext(ctx, contract.Authentication("missing session cookie"))
	}
	db, err := h.db()
	if err != nil {
		return nil, contract.WithContext(ctx, contract.Internal("admin database is not attached"))
	}
	grants, err := auth.HasPermissions(ctx, db, user, keys...)
	if err != nil {
		return nil, contract.WithContext(ctx, contract.Internal("check admin permissions"))
	}
	return grants, nil
}

func (h *handlers) requirePermission(ctx context.Context, key string) error {
	grants, err := h.permissionGrants(ctx, key)
	if err != nil {
		return err
	}
	if !grants[key] {
		return contract.WithContext(ctx, contract.Authorization("admin permission denied"))
	}
	return nil
}

func permissionKeys(models []*registered) []string {
	keys := make([]string, 0, len(models)*4)
	for _, m := range models {
		keys = append(keys,
			m.meta.Permissions.View,
			m.meta.Permissions.Create,
			m.meta.Permissions.Update,
			m.meta.Permissions.Delete,
		)
	}
	return keys
}

func capabilities(m *registered, grants map[string]bool) Capabilities {
	return Capabilities{
		View:   grants[m.meta.Permissions.View] && (m.actions.List || m.actions.Detail),
		Create: grants[m.meta.Permissions.Create] && m.actions.Create,
		Update: grants[m.meta.Permissions.Update] && m.actions.Update,
		Delete: grants[m.meta.Permissions.Delete] && m.actions.Delete,
	}
}
