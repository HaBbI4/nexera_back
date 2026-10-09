package auth

import (
	"fmt"
	"time"

	"nexera_back/internal/models"

	"github.com/golang-jwt/jwt/v5"
)

// Claims представляет данные, зашитые в JWT токен админ-панели.
type Claims struct {
	UserID   int    `json:"user_id"`
	RoleCode string `json:"role_code"`
	jwt.RegisteredClaims
}

// GenerateToken выпускает подписанный JWT для пользователя админ-панели.
// Ожидает, что user.Role уже загружен (иначе RoleCode в токене будет пустым).
func GenerateToken(user *models.User, secret string, ttl time.Duration) (string, error) {
	var roleCode string
	if user.Role != nil {
		roleCode = user.Role.Code
	}

	claims := Claims{
		UserID:   user.ID,
		RoleCode: roleCode,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.Email,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseToken проверяет подпись и срок действия токена и возвращает его claims.
func ParseToken(tokenString, secret string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}
