package repository

import (
	"database/sql"

	"durpdeploy/internal/db"
)

type Repository struct {
	DB      *sql.DB
	Queries *db.Queries
}

func New(dbConn *sql.DB) *Repository {
	return &Repository{
		DB:      dbConn,
		Queries: db.New(dbConn),
	}
}
