package db

import "database/sql"

func GetDB(dbsource string) (*sql.DB, error) {
	return sql.Open("sqlite3", dbsource)
}
