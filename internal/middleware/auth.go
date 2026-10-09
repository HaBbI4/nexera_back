package middleware

import (
	"context"
	"net/http"
	"slices"
	"strings"

	"nexera_back/internal/auth"
	"nexera_back/internal/httpresponse"
)

type contextKey string

const userContextKey contextKey = "auth_user"

// AuthUser представляет данные пользователя админ-панели, извлеченные из JWT токена.
type AuthUser struct {
	ID       int
	RoleCode string
}

// RequireAuth проверяет JWT из заголовка "Authorization: Bearer <token>" и кладет
// данные пользователя в контекст запроса. Применяется ко всем защищенным /api/v1/admin/* маршрутам.
func RequireAuth(jwtSecret string) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			tokenString, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || tokenString == "" {
				httpresponse.WriteError(w, http.StatusUnauthorized, "missing bearer token")
				return
			}

			claims, err := auth.ParseToken(tokenString, jwtSecret)
			if err != nil {
				httpresponse.WriteError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			user := &AuthUser{ID: claims.UserID, RoleCode: claims.RoleCode}
			next(w, r.WithContext(context.WithValue(r.Context(), userContextKey, user)))
		}
	}
}

// RequireRole проверяет, что аутентифицированный пользователь имеет одну из переданных ролей.
// Должен применяться после RequireAuth (иначе в контексте не будет пользователя).
func RequireRole(codes ...string) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user, ok := UserFromContext(r.Context())
			if !ok {
				httpresponse.WriteError(w, http.StatusUnauthorized, "missing bearer token")
				return
			}

			if !slices.Contains(codes, user.RoleCode) {
				httpresponse.WriteError(w, http.StatusForbidden, "insufficient permissions")
				return
			}

			next(w, r)
		}
	}
}

// UserFromContext возвращает пользователя, положенного в контекст запроса middleware RequireAuth.
func UserFromContext(ctx context.Context) (*AuthUser, bool) {
	user, ok := ctx.Value(userContextKey).(*AuthUser)
	return user, ok
}
