package middlewares

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"user_service/internal/app"
	"user_service/internal/domain"
	"user_service/internal/transport/http/presenters"

	"github.com/gin-gonic/gin"
)

const (
	// ContextUserKey is used to store authenticated user in gin.Context.
	ContextUserKey = "current_user"
	adminRole      = "ADMIN"
)

type ssoAuthenticateResponse struct {
	FirstName  string   `json:"first_name"`
	LastName   string   `json:"last_name"`
	Email      string   `json:"email"`
	LocaleType *string  `json:"locale_type"`
	Roles      []string `json:"roles"`
}

// AdminOnly ensures that the request is performed by a user that has admin privileges.
func AdminOnly(a *app.App) gin.HandlerFunc {
	defaultAdmins := buildAdminLookup(a.Config.DefaultAdminEmails)
	httpClient := &http.Client{Timeout: 5 * time.Second}

	return func(ctx *gin.Context) {
		token, err := extractBearerToken(ctx.GetHeader("Authorization"))
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, presenters.Error(err))
			return
		}

		user, authErr := authenticateAdminViaSSO(ctx, httpClient, a.Config.SSOAuthenticateURL, token)
		if authErr != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, presenters.Error(fmt.Errorf("failed to authenticate via sso: %w", authErr)))
			return
		}

		if !userHasAdminPrivileges(user, defaultAdmins) {
			ctx.AbortWithStatusJSON(http.StatusForbidden, presenters.Error(fmt.Errorf("admin access required")))
			return
		}

		ctx.Set(ContextUserKey, user)
		ctx.Next()
	}
}

func authenticateAdminViaSSO(ctx *gin.Context, client *http.Client, authenticateURL, token string) (*domain.User, error) {
	req, err := http.NewRequestWithContext(ctx.Request.Context(), http.MethodGet, authenticateURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build sso request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request sso authenticate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sso authenticate returned status %d", resp.StatusCode)
	}

	var payload ssoAuthenticateResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode sso authenticate response: %w", err)
	}

	return &domain.User{
		FirstName:  payload.FirstName,
		LastName:   payload.LastName,
		Email:      payload.Email,
		LocaleType: payload.LocaleType,
		Roles:      payload.Roles,
	}, nil
}

func extractBearerToken(header string) (string, error) {
	if header == "" {
		return "", fmt.Errorf("missing Authorization header")
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", fmt.Errorf("invalid Authorization header format")
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", fmt.Errorf("invalid Authorization header format")
	}
	return token, nil
}

func userHasAdminPrivileges(user *domain.User, defaultAdmins map[string]struct{}) bool {
	if user == nil {
		return false
	}

	if len(user.Roles) > 0 {
		for _, role := range user.Roles {
			if strings.EqualFold(role, adminRole) {
				return true
			}
		}
	}

	if len(defaultAdmins) == 0 {
		return false
	}

	normalized := normalizeEmail(user.Email)
	_, ok := defaultAdmins[normalized]
	return ok
}

func buildAdminLookup(emails []string) map[string]struct{} {
	lookup := make(map[string]struct{}, len(emails))
	for _, email := range emails {
		if normalized := normalizeEmail(email); normalized != "" {
			lookup[normalized] = struct{}{}
		}
	}
	return lookup
}

func normalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}
