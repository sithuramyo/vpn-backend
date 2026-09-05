package database

import (
	"fmt"
	"io/fs"
	"log"
	"sort"
	"strings"

	"gorm.io/gorm"
)

// migrationsFS is set by SetMigrationsFS in main, pointing at the
// migrations/ directory (e.g. via os.DirFS or an embedded FS).
var migrationsFS fs.FS

func SetMigrationsFS(f fs.FS) {
	migrationsFS = f
}

type migrationRecord struct {
	Version string `gorm:"primaryKey;column:version"`
}

func (migrationRecord) TableName() string { return "schema_migrations" }

// Migrate applies all pending *.up.sql files from the configured migrations
// filesystem, in filename order, tracking applied versions in
// schema_migrations so re-runs are idempotent.
func Migrate(db *gorm.DB) error {
	if migrationsFS == nil {
		return fmt.Errorf("migrations filesystem not configured")
	}

	if err := db.AutoMigrate(&migrationRecord{}); err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	entries, err := fs.ReadDir(migrationsFS, ".")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var upFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			upFiles = append(upFiles, e.Name())
		}
	}
	sort.Strings(upFiles)

	var applied []migrationRecord
	if err := db.Find(&applied).Error; err != nil {
		return fmt.Errorf("load applied migrations: %w", err)
	}
	appliedSet := make(map[string]bool, len(applied))
	for _, a := range applied {
		appliedSet[a.Version] = true
	}

	for _, name := range upFiles {
		version := strings.TrimSuffix(name, ".up.sql")
		if appliedSet[version] {
			continue
		}

		contents, err := fs.ReadFile(migrationsFS, name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		err = db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(string(contents)).Error; err != nil {
				return fmt.Errorf("apply migration %s: %w", name, err)
			}
			return tx.Create(&migrationRecord{Version: version}).Error
		})
		if err != nil {
			return err
		}

		log.Printf("applied migration %s", version)
	}

	return nil
}
