package middlewares

import (
	"fmt"
	"strings"
	"game_server/internal/app"
	"game_server/internal/domain"

	"github.com/gin-gonic/gin"
)

const (
	// ContextUserKey is used to store authenticated user in gin.Context.
	ContextUserKey = "current_user"
	adminRole      = "ADMIN"
)

// AdminOnly ensures that the request is performed by a user that has admin privileges.
// !TODO For admin roles (check JWT)
func AdminOnly(a *app.App) gin.HandlerFunc {
	// defaultAdmins := buildAdminLookup(a.Config.DefaultAdminEmails)

	return func(ctx *gin.Context) {
		// token, err := extractBearerToken(ctx.GetHeader("Authorization"))
		// if err != nil {
		// 	ctx.AbortWithStatusJSON(http.StatusUnauthorized, presenters.Error(err))
		// 	return
		// }

		// if !userHasAdminPrivileges(user, defaultAdmins) {
		// 	ctx.AbortWithStatusJSON(http.StatusForbidden, presenters.Error(fmt.Errorf("admin access required")))
		// 	return
		// }

		// ctx.Set(ContextUserKey, user)
		ctx.Next()
	}
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
