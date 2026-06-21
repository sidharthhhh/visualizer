package db

import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigrations(dsn string, logger *slog.Logger) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("opening db for migrations: %w", err)
	}
	defer db.Close()

	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		logger.Warn("could not create migrator, skipping", "error", err)
		return nil
	}
	defer m.Close()

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		logger.Warn("could not get migration version", "error", err)
	}

	if dirty {
		logger.Warn("dirty database detected, forcing version", "version", version)
		if err := m.Force(int(version)); err != nil {
			logger.Warn("could not force version", "error", err)
		}
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		logger.Warn("migration error", "error", err)
	}

	version, dirty, err = m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		logger.Warn("could not get final migration version", "error", err)
	}

	logger.Info("migrations complete", "version", version, "dirty", dirty)
	return nil
}
