// Package testutil provides an in-memory SQLite-backed database for unit
// tests, so services/middleware can be exercised without a live
// PostgreSQL instance. Production always uses PostgreSQL (see
// internal/database); this is test-only scaffolding.
package testutil

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"vpn-backend/internal/models"
)

func NewTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	err = db.AutoMigrate(
		&models.Admin{},
		&models.VPNUser{},
		&models.VPNDevice{},
		&models.VPNServer{},
		&models.AccessKey{},
		&models.AuditLog{},
		&models.ServerMetric{},
	)
	if err != nil {
		t.Fatalf("automigrate test db: %v", err)
	}

	return db
}
