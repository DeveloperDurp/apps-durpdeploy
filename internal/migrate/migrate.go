package migrate

import (
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"

	"durpdeploy/migrations"
)

func Run(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	goose.SetBaseFS(migrations.FS)
	goose.SetDialect("sqlite3")

	if err := goose.Up(db, "."); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return db, nil
}
