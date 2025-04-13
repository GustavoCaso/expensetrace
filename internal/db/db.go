package db

import (
	"database/sql"
	// import sqlite driver.
	_ "github.com/mattn/go-sqlite3"
)

func GetDB(dbsource string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbsource)
	if err != nil {
		return nil, err
	}

	// Enable foreign key constraints
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		return nil, err
	}

	return db, nil
}
