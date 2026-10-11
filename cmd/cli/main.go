// Command cli — служебные команды для обслуживания проекта (аналог php artisan).
//
// Использование:
//
//	go run ./cmd/cli create-admin --email admin@example.com [--name "Администратор"]
//
// Пароль берется из переменной окружения ADMIN_PASSWORD, а если она не задана —
// запрашивается в терминале без эха (дважды, для подтверждения).
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"golang.org/x/term"

	"nexera_back/internal/config"
	"nexera_back/internal/database"
	"nexera_back/internal/logger"
	"nexera_back/internal/models"
	"nexera_back/internal/repository"
	"nexera_back/internal/services"
)

const usage = `Использование: cli <команда> [флаги]

Команды:
  create-admin  создать пользователя админ-панели с ролью admin
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "create-admin":
		err = createAdmin(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "неизвестная команда %q\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "ошибка: %v\n", err)
		os.Exit(1)
	}
}

// createAdmin создает пользователя с ролью admin через UserService.Create — с той же
// валидацией email/пароля и bcrypt, что и POST /api/v1/admin/users.
func createAdmin(args []string) error {
	fs := flag.NewFlagSet("create-admin", flag.ExitOnError)
	email := fs.String("email", "", "email администратора (обязательно)")
	name := fs.String("name", "Администратор", "имя администратора")
	fs.Parse(args)

	if *email == "" {
		fs.Usage()
		return errors.New("--email is required")
	}

	password, err := readPassword()
	if err != nil {
		return err
	}

	cfg := config.Load()
	log := logger.New(&cfg.Logger)

	db, err := database.Connect(&cfg.Database, log)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	roleService := services.NewRoleService(repository.NewRoleRepository(db), log)
	userService := services.NewUserService(repository.NewUserRepository(db), log)

	role, err := roleService.GetByCode(models.RoleCodeAdmin)
	if err != nil {
		if errors.Is(err, repository.ErrRoleNotFound) {
			return errors.New("role admin not found, run migrations first (make migrate-up)")
		}
		return err
	}

	user, err := userService.Create(models.CreateUserRequest{
		RoleID:   role.ID,
		Name:     *name,
		Email:    *email,
		Password: password,
	})
	if err != nil {
		if errors.Is(err, repository.ErrUserEmailTaken) {
			return fmt.Errorf("user with email %s already exists", *email)
		}
		return err
	}

	fmt.Printf("Администратор создан: id=%d, email=%s\n", user.ID, user.Email)
	return nil
}

// readPassword берет пароль из ADMIN_PASSWORD (для неинтерактивного запуска) или
// запрашивает его в терминале. Длину и прочие правила проверяет UserService.
func readPassword() (string, error) {
	if password := os.Getenv("ADMIN_PASSWORD"); password != "" {
		return password, nil
	}

	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return "", errors.New("stdin is not a terminal: set ADMIN_PASSWORD or run interactively")
	}

	fmt.Fprint(os.Stderr, "Пароль: ")
	password, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}

	fmt.Fprint(os.Stderr, "Повторите пароль: ")
	confirm, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}

	if string(password) != string(confirm) {
		return "", errors.New("passwords do not match")
	}

	return string(password), nil
}
