package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/LAA-Software-Engineering/gombit/config"
	"github.com/danielgtaylor/huma/v2"
	"gorm.io/gorm"
)

const bearerSecurityName = "bearerAuth"

// Mount registers framework-owned auth Huma routes on api:
// POST register, login, refresh, logout and GET /me.
func Mount(api huma.API, db *gorm.DB, cfg config.Config) error {
	if api == nil {
		return errors.New("auth: nil API")
	}
	if !cfg.Auth.Enabled() {
		return nil
	}
	svc, err := NewService(db, cfg)
	if err != nil {
		return err
	}
	installBearerScheme(api)
	prefix := strings.TrimSuffix(cfg.API.Prefix, "/")
	if prefix == "" {
		prefix = "/api/v1"
	}

	huma.Register(api, huma.Operation{
		OperationID: "register",
		Method:      http.MethodPost,
		Path:        prefix + "/auth/register",
		Summary:     "Register a user",
		Tags:        []string{"Auth"},
	}, svc.register)

	huma.Register(api, huma.Operation{
		OperationID: "login",
		Method:      http.MethodPost,
		Path:        prefix + "/auth/login",
		Summary:     "Log in and issue Bearer tokens",
		Tags:        []string{"Auth"},
	}, svc.login)

	huma.Register(api, huma.Operation{
		OperationID: "refresh",
		Method:      http.MethodPost,
		Path:        prefix + "/auth/refresh",
		Summary:     "Rotate the refresh token and issue a new access token",
		Tags:        []string{"Auth"},
	}, svc.refresh)

	huma.Register(api, huma.Operation{
		OperationID: "logout",
		Method:      http.MethodPost,
		Path:        prefix + "/auth/logout",
		Summary:     "Revoke the current refresh token",
		Tags:        []string{"Auth"},
	}, svc.logout)

	huma.Register(api, huma.Operation{
		OperationID: "get-me",
		Method:      http.MethodGet,
		Path:        prefix + "/me",
		Summary:     "Return the authenticated user",
		Tags:        []string{"Auth"},
		Security:    []map[string][]string{{bearerSecurityName: {}}},
		Middlewares: huma.Middlewares{svc.requireBearer()},
	}, svc.me)

	return nil
}

func installBearerScheme(api huma.API) {
	openapi := api.OpenAPI()
	if openapi.Components == nil {
		openapi.Components = &huma.Components{}
	}
	if openapi.Components.SecuritySchemes == nil {
		openapi.Components.SecuritySchemes = map[string]*huma.SecurityScheme{}
	}
	openapi.Components.SecuritySchemes[bearerSecurityName] = &huma.SecurityScheme{
		Type:         "http",
		Scheme:       "bearer",
		BearerFormat: "JWT",
		Description:  "Short-lived access JWT. Send `Authorization: Bearer <token>`. Hold the token in memory only.",
	}
}
