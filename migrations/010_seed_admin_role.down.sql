-- Не откатится, пока роль назначена пользователям (users.role_id ON DELETE RESTRICT) —
-- сначала нужно удалить или переназначить их.
DELETE FROM roles WHERE code = 'admin';
