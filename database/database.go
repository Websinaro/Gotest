package database

import (
	"database/sql"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"

	"kabackend/config"
)

var DB *sql.DB

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

// Connect opens the Postgres connection pool (equivalent of
// sqlalchemy.create_engine + sessionmaker) and stores it in DB.
//
// Pool limits mirror SQLAlchemy's defaults (pool_size=5, max_overflow=10,
// i.e. 15 connections open at most) so the Go service can't fan out to more
// Postgres connections under load than the Python service ever did.
// Override via DB_MAX_OPEN_CONNS / DB_MAX_IDLE_CONNS / DB_CONN_MAX_LIFETIME_MIN
// if a deployment needs different limits.
func Connect() {
	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	maxOpen := getEnvInt("DB_MAX_OPEN_CONNS", 15) // SQLAlchemy pool_size(5) + max_overflow(10)
	maxIdle := getEnvInt("DB_MAX_IDLE_CONNS", 5)  // SQLAlchemy pool_size
	maxLifetimeMin := getEnvInt("DB_CONN_MAX_LIFETIME_MIN", 30)

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(time.Duration(maxLifetimeMin) * time.Minute)

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	DB = db
}

// createTableStatements mirrors Base.metadata.create_all(bind=engine) in
// main.py: every table is created if it doesn't already exist.
var createTableStatements = []string{
	`CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		email VARCHAR(150) NOT NULL UNIQUE,
		phone VARCHAR(20) NOT NULL,
		password VARCHAR(255) NOT NULL,
		district VARCHAR(50) NOT NULL,
		role VARCHAR(40) NOT NULL DEFAULT 'user',
		created_time VARCHAR(50) NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS safety_contacts (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id),
		name VARCHAR(100) NOT NULL,
		relationship VARCHAR(50),
		phone VARCHAR(20) NOT NULL,
		email VARCHAR(150),
		address VARCHAR(255),
		created_time VARCHAR(50) NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS sos_alerts (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id),
		latitude DOUBLE PRECISION NOT NULL,
		longitude DOUBLE PRECISION NOT NULL,
		status VARCHAR(20) NOT NULL DEFAULT 'active',
		message VARCHAR(255),
		created_time VARCHAR(50) NOT NULL,
		resolved_time VARCHAR(50)
	)`,
	`CREATE TABLE IF NOT EXISTS device_tokens (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id),
		fcm_token VARCHAR(255) NOT NULL UNIQUE,
		platform VARCHAR(20),
		updated_time VARCHAR(50) NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS notifications (
		id SERIAL PRIMARY KEY,
		title VARCHAR(150) NOT NULL,
		message VARCHAR(1000) NOT NULL,
		severity VARCHAR(20) NOT NULL DEFAULT 'orange',
		district VARCHAR(50),
		created_by INTEGER NOT NULL REFERENCES users(id),
		created_by_name VARCHAR(100) NOT NULL,
		active BOOLEAN NOT NULL DEFAULT TRUE,
		created_time VARCHAR(50) NOT NULL,
		updated_time VARCHAR(50) NOT NULL
	)`,
}

// indexStatements mirrors alembic migrations 0001_performance_indexes and
// 0002_notifications_table's index creation, using IF NOT EXISTS so it's
// safe to run every startup.
var indexStatements = []string{
	`CREATE INDEX IF NOT EXISTS ix_users_phone ON users (phone)`,
	`CREATE INDEX IF NOT EXISTS ix_users_email ON users (email)`,
	`CREATE INDEX IF NOT EXISTS ix_safety_contacts_user_id ON safety_contacts (user_id)`,
	`CREATE INDEX IF NOT EXISTS ix_safety_contacts_phone ON safety_contacts (phone)`,
	`CREATE INDEX IF NOT EXISTS ix_safety_contacts_email ON safety_contacts (email)`,
	`CREATE INDEX IF NOT EXISTS ix_sos_alerts_user_id ON sos_alerts (user_id)`,
	`CREATE INDEX IF NOT EXISTS ix_sos_alerts_status ON sos_alerts (status)`,
	`CREATE INDEX IF NOT EXISTS ix_sos_alerts_user_status ON sos_alerts (user_id, status)`,
	`CREATE INDEX IF NOT EXISTS ix_device_tokens_user_id ON device_tokens (user_id)`,
	`CREATE INDEX IF NOT EXISTS ix_notifications_district ON notifications (district)`,
	`CREATE INDEX IF NOT EXISTS ix_notifications_created_by ON notifications (created_by)`,
}

// RunMigrations creates tables and indexes if missing. Combines the role
// main.py gave to both run_migrations() (alembic upgrade head) and
// Base.metadata.create_all(bind=engine).
func RunMigrations() {
	for _, stmt := range createTableStatements {
		if _, err := DB.Exec(stmt); err != nil {
			log.Fatalf("failed to create table: %v", err)
		}
	}
	for _, stmt := range indexStatements {
		if _, err := DB.Exec(stmt); err != nil {
			log.Fatalf("failed to create index: %v", err)
		}
	}
}
