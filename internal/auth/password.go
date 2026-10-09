package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword возвращает bcrypt-хеш пароля для хранения в users.password_hash.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword сравнивает пароль с bcrypt-хешем. Возвращает ошибку, если они не совпадают.
func CheckPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
