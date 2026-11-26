package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/migrate"

	"github.com/nexus-rmm/backend/internal/config"
)

type Database struct {
	*bun.DB
}

func New(cfg *config.DatabaseConfig) (*Database, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.SSLMode)

	connector := pgdriver.NewConnector(
		pgdriver.WithDSN(dsn),
	)

	sqldb := sql.OpenDB(connector)
	sqldb.SetMaxOpenConns(int(cfg.MaxConnections))
	sqldb.SetMaxIdleConns(int(cfg.MaxIdleConns))
	sqldb.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	db := bun.NewDB(sqldb, pgdialect.New())

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Database{DB: db}, nil
}

func (d *Database) Close() error {
	return d.DB.Close()
}

func (d *Database) Migrate(migrations *migrate.Migrations) error {
	migrator := migrate.NewMigrator(d.DB, migrations)
	_, err := migrator.Migrate(context.Background())
	return err
}

func (d *Database) MigrateWithContext(ctx context.Context, migrations *migrate.Migrations) error {
	migrator := migrate.NewMigrator(d.DB, migrations)
	_, err := migrator.Migrate(ctx)
	return err
}

func (d *Database) MigrateTo(ctx context.Context, migrations *migrate.Migrations, name string) error {
	migrator := migrate.NewMigrator(d.DB, migrations)
	return migrator.RunMigration(ctx, name)
}

func (d *Database) CreateMigration(migrations *migrate.Migrations, name string) error {
	migrator := migrate.NewMigrator(d.DB, migrations)
	_, err := migrator.CreateGoMigration(context.Background(), name)
	return err
}

func (d *Database) HealthCheck(ctx context.Context) error {
	return d.Ping()
}

func (d *Database) Stats() interface{} {
	return d.DB.DBStats()
}