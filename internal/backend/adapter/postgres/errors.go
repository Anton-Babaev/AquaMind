package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5"
)

// isNoRows сообщает, что ошибка — это «нет строк» (pgx.ErrNoRows).
// Используется в SaveBatch: при ON CONFLICT DO NOTHING ... RETURNING id
// отсутствие строки означает, что тик уже был сохранён (дубликат).
func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
