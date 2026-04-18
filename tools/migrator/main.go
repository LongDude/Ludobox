package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	postgresmigrate "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

func main() {
	var (
		user, password, host, port, dbname, migrationsPath, migrationsTable string
	)

	flag.StringVar(&user, "user", "", "PostgreSQL username")
	flag.StringVar(&password, "password", "", "PostgreSQL password")
	flag.StringVar(&host, "host", "localhost", "PostgreSQL host")
	flag.StringVar(&port, "port", "5432", "PostgreSQL port")
	flag.StringVar(&dbname, "dbname", "", "PostgreSQL database name")
	flag.StringVar(&migrationsPath, "migrations-path", "db/migrations", "Path to shared migration files")
	flag.StringVar(&migrationsTable, "migrations-table", "schema_migrations", "Migration metadata table name")
	flag.Parse()

	if user == "" || password == "" || host == "" || port == "" || dbname == "" {
		log.Fatal("missing required database connection parameters")
	}

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user,
		password,
		host,
		port,
		dbname,
	)

	if err := runMigrations(dsn, migrationsPath, migrationsTable); err != nil {
		log.Fatalf("shared migration failed: %v", err)
	}

	log.Println("shared migrations applied successfully")
}

func runMigrations(dsn string, migrationsPath string, migrationsTable string) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open database connection: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	driver, err := postgresmigrate.WithInstance(db, &postgresmigrate.Config{
		MigrationsTable: migrationsTable,
	})
	if err != nil {
		return fmt.Errorf("create postgres driver: %w", err)
	}

	absPath, err := filepath.Abs(migrationsPath)
	if err != nil {
		return fmt.Errorf("resolve migrations path: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", filepath.ToSlash(absPath)),
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("initialize migrator: %w", err)
	}

	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			log.Printf("source close error: %v", srcErr)
		}
		if dbErr != nil && !errors.Is(dbErr, sql.ErrConnDone) {
			log.Printf("database close error: %v", dbErr)
		}
	}()

	version, dirty, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("read current version: %w", err)
	}
	if dirty {
		return fmt.Errorf(
			"database is dirty at version %d; resolve state in table %s before continuing",
			version,
			migrationsTable,
		)
	}

	log.Printf("applying shared migrations from %s", absPath)

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Printf("no new shared migrations to apply")
			return nil
		}
		return fmt.Errorf("apply migrations: %w", err)
	}

	log.Printf("shared migrations applied successfully")
	return nil
}
