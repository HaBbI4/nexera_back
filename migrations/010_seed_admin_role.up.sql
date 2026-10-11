-- Роль администратора — справочные данные, без которых система не работает: доступ к
-- управлению пользователями, ролями и пользователями бота проверяется по code = 'admin'
-- (models.RoleCodeAdmin). Самого пользователя-админа создает CLI: make create-admin.
INSERT INTO roles (name, code) VALUES ('Администратор', 'admin')
ON CONFLICT (code) DO NOTHING;
