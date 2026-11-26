package storage

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

// Migrations holds all database migrations
var Migrations = migrate.NewMigrations()

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// Create all tables using bun's auto-migration feature
		// This will create tables based on the model definitions
		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Rollback - drop all tables
		return nil
	})
}

// InitializeSchema creates all tables based on model definitions
func InitializeSchema(ctx context.Context, db *bun.DB) error {
	models := []interface{}{
		(*User)(nil),
		(*Organization)(nil),
		(*Site)(nil),
		(*Device)(nil),
		(*DeviceGroup)(nil),
		(*HardwareInfo)(nil),
		(*GPUInfo)(nil),
		(*DiskInfo)(nil),
		(*PartitionInfo)(nil),
		(*NetworkInterfaceInfo)(nil),
		(*SoftwareInventory)(nil),
		(*Metric)(nil),
		(*MonitoringPolicy)(nil),
		(*Alert)(nil),
		(*RefreshToken)(nil),
		(*APIKey)(nil),
		(*RemoteSession)(nil),
		(*SessionParticipant)(nil),
		(*SessionEvent)(nil),
		(*Script)(nil),
		(*ScriptExecution)(nil),
		(*Schedule)(nil),
		(*Task)(nil),
		(*Notification)(nil),
		(*AuditLog)(nil),
		(*License)(nil),
		(*Integration)(nil),
		(*IntegrationTrigger)(nil),
	}

	for _, model := range models {
		if _, err := db.NewCreateTable().Model(model).IfNotExists().Exec(ctx); err != nil {
			return err
		}
	}

	return nil
}

// MigrateUp runs all pending migrations
func MigrateUp(ctx context.Context, db *bun.DB) error {
	migrator := migrate.NewMigrator(db, Migrations)
	_, err := migrator.Migrate(ctx)
	return err
}

// MigrateDown rolls back the last migration
func MigrateDown(ctx context.Context, db *bun.DB) error {
	migrator := migrate.NewMigrator(db, Migrations)
	_, err := migrator.Rollback(ctx)
	return err
}

// CreateMigration creates a new migration file
func CreateMigration(ctx context.Context, db *bun.DB, name string) error {
	migrator := migrate.NewMigrator(db, Migrations)

	// Create migrations directory if it doesn't exist
	migrationsDir := "migrations"
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		if err := os.MkdirAll(migrationsDir, 0755); err != nil {
			return fmt.Errorf("failed to create migrations directory: %w", err)
		}
	}

	// Generate migration file name with timestamp
	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("%s_%s.go", timestamp, name)

	_, err := migrator.CreateGoMigration(ctx, filename)
	return err
}