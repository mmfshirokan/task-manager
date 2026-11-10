package schema

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgx/v5/stdlib" // регистрирует драйвер database/sql с именем "pgx"
)

// Migrate применяет к базе по адресу dsn все ещё не применённые миграции и
// возвращает их версии.
func Migrate(ctx context.Context, dsn string) (_ []int64, err error) {
	migrations, err := fs.Sub(Fs, "migrations")
	if err != nil {
		return nil, fmt.Errorf("open migrations directory: %w", err)
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close database: %w", closeErr))
		}
	}()

	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations)
	if err != nil {
		return nil, fmt.Errorf("create migration provider: %w", err)
	}

	results, err := provider.Up(ctx)
	if err != nil {
		return nil, fmt.Errorf("apply migrations: %w", err)
	}

	applied := make([]int64, 0, len(results))
	for _, r := range results {
		applied = append(applied, r.Source.Version)
	}

	return applied, nil
}
