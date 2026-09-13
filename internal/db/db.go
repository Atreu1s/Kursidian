package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

const driverName = "sqlite"

func Open(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)", path)

	handle, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("open database %q: %w", path, err)
	}

	handle.SetMaxOpenConns(1)

	if err := handle.Ping(); err != nil {
		handle.Close()
		return nil, fmt.Errorf("ping database %q: %w", path, err)
	}

	return handle, nil
}
