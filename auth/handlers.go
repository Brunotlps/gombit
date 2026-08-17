package auth

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/LAA-Software-Engineering/gombit/contract"
	"github.com/danielgtaylor/huma/v2"
)

type credentialsBody struct {
	Email    string `json:"email" minLength:"1" maxLength:"255" format:"email" example:"ada@example.com" doc:"Account email"`
	Password string `json:"password" minLength:"8" maxLength:"72" example:"correct-horse" doc:"Account password"`
}

type loginBody struct {
	Email    string `json:"email" minLength:"1" maxLength:"255" format:"email" example:"ada@example.com" doc:"Account email"`
	Password string `json:"password" minLength:"1" maxLength:"72" example:"correct-horse" doc:"Account password"`
}

type refreshBody struct {
	RefreshToken string `json:"refresh_token" minLength:"1" example:"aa11..." doc:"Current refresh token"`
}

type registerInput struct {
	Body credentialsBody
}

type registerOutput struct {
	Body contract.Data[PublicUser]
}

type loginInput struct {
	Body loginBody
}

type tokenOutput struct {
	Body contract.Data[TokenPair]
}

type refreshInput struct {
	Body refreshBody
}

type logoutInput struct {
	Body refreshBody
}

type logoutResult struct {
	OK bool `json:"ok" example:"true" doc:"True when the refresh token was revoked or already invalid"`
}

type logoutOutput struct {
	Body contract.Data[logoutResult]
}

type meOutput struct {
	Body contract.Data[PublicUser]
}

func (s *Service) register(ctx context.Context, input *registerInput) (*registerOutput, error) {
	user, err := s.Register(ctx, input.Body.Email, input.Body.Password)
	if err != nil {
		return nil, mapServiceError(ctx, err)
	}
	return &registerOutput{Body: contract.Data[PublicUser]{Data: toPublicUser(user)}}, nil
}

func (s *Service) login(ctx context.Context, input *loginInput) (*tokenOutput, error) {
	user, err := s.Authenticate(ctx, input.Body.Email, input.Body.Password)
	if err != nil {
		return nil, mapServiceError(ctx, err)
	}
	pair, err := s.IssueTokens(ctx, user)
	if err != nil {
		return nil, contract.WithContext(ctx, contract.Internal("issue tokens"))
	}
	return &tokenOutput{Body: contract.Data[TokenPair]{Data: pair}}, nil
}

func (s *Service) refresh(ctx context.Context, input *refreshInput) (*tokenOutput, error) {
	pair, err := s.RotateRefresh(ctx, input.Body.RefreshToken)
	if err != nil {
		return nil, mapServiceError(ctx, err)
	}
	return &tokenOutput{Body: contract.Data[TokenPair]{Data: pair}}, nil
}

func (s *Service) logout(ctx context.Context, input *logoutInput) (*logoutOutput, error) {
	if err := s.RevokeRefresh(ctx, input.Body.RefreshToken); err != nil {
		return nil, mapServiceError(ctx, err)
	}
	return &logoutOutput{Body: contract.Data[logoutResult]{Data: logoutResult{OK: true}}}, nil
}

func (s *Service) me(ctx context.Context, _ *struct{}) (*meOutput, error) {
	user, ok := UserFromContext(ctx)
	if !ok {
		return nil, contract.WithContext(ctx, contract.Authentication("missing bearer token"))
	}
	return &meOutput{Body: contract.Data[PublicUser]{Data: toPublicUser(user)}}, nil
}

func (s *Service) requireBearer() func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		header := ctx.Header("Authorization")
		raw, ok := bearerToken(header)
		if !ok {
			writeAuthError(ctx, contract.WithContext(ctx.Context(), contract.Authentication("missing bearer token")))
			return
		}
		user, err := s.ParseAccess(ctx.Context(), raw)
		if err != nil {
			writeAuthError(ctx, contract.WithContext(ctx.Context(), contract.Authentication("invalid access token")))
			return
		}
		next(huma.WithValue(ctx, userContextKey{}, user))
	}
}

func bearerToken(header string) (string, bool) {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" {
		return "", false
	}
	return token, true
}

func writeAuthError(ctx huma.Context, env *contract.ErrorEnvelope) {
	if env == nil {
		return
	}
	ctx.SetHeader("Content-Type", "application/json")
	ctx.SetHeader("WWW-Authenticate", `Bearer realm="api"`)
	ctx.SetStatus(env.GetStatus())
	_ = json.NewEncoder(ctx.BodyWriter()).Encode(env)
}

func mapServiceError(ctx context.Context, err error) error {
	switch {
	case errors.Is(err, errInvalidCredentials):
		return contract.WithContext(ctx, contract.Authentication("invalid email or password"))
	case errors.Is(err, errInvalidRefreshToken), errors.Is(err, errRefreshReuse):
		return contract.WithContext(ctx, contract.Authentication("invalid refresh token"))
	case errors.Is(err, errInvalidAccessToken), errors.Is(err, errMissingBearer):
		return contract.WithContext(ctx, contract.Authentication("invalid access token"))
	case errors.Is(err, errEmailTaken):
		return contract.WithContext(ctx, contract.Conflict("email already registered"))
	case errors.Is(err, errPasswordTooLong), errors.Is(err, errPasswordRequired):
		return contract.WithContext(ctx, contract.Validation("The request contains invalid fields.", map[string][]string{
			"password": {err.Error()},
		}))
	case errors.Is(err, errUserNotFound):
		return contract.WithContext(ctx, contract.Authentication("invalid access token"))
	default:
		return contract.WithContext(ctx, contract.Internal("auth request failed"))
	}
}

type userContextKey struct{}

// UserFromContext returns the authenticated user stored by requireBearer.
func UserFromContext(ctx context.Context) (User, bool) {
	if ctx == nil {
		return User{}, false
	}
	user, ok := ctx.Value(userContextKey{}).(User)
	return user, ok
}
