package db

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func GetDB(dbsource string) (*sql.DB, error) {
	return sql.Open("sqlite3", dbsource)
}
