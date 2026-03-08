package database

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/conf"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Init() {
	cfg := conf.Read()

	newLogger := logger.New(
		log.New(),
		logger.Config{
			//SlowThreshold:             time.Second,  // Slow SQL threshold
			LogLevel:                  logger.Warn, // Log level
			IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
			//ParameterizedQueries:      true,         // Don't include params in the SQL log
			Colorful: true, // Disable color
		},
	)

	// Choose driver.
	var dialector gorm.Dialector
	switch os.Getenv("DB_ADAPTER") {
	case "mysql":
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Europe/Berlin", os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), os.Getenv("DB_PORT"))
		dialector = mysql.New(mysql.Config{DSN: dsn})
	case "postgres":
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Europe/Berlin", os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), os.Getenv("DB_PORT"))
		dialector = postgres.New(postgres.Config{DSN: dsn})
	default:
		// SQLite3 is a single-writer database. For production multi-user
		// workloads use PostgreSQL (set DB_ADAPTER=postgres).
		// For development: WAL mode + busy_timeout makes concurrent access
		// tolerable for a small number of users.
		sqlite_vec.Auto()
		// _busy_timeout: retry writes for up to 10 s before returning SQLITE_BUSY.
		// _journal_mode=WAL: readers never block writers and vice-versa.
		// _synchronous=NORMAL: safe durability with WAL, faster than FULL.
		dsn := cfg.DbFileName + "?_busy_timeout=10000&_journal_mode=WAL&_synchronous=NORMAL"
		dialector = sqlite.Open(dsn)
	}

	/// Open and assign database.
	config := &gorm.Config{
		Logger:                                   newLogger,
		DisableForeignKeyConstraintWhenMigrating: false, // Enable foreign key constraints for data integrity
	}
	db, err := gorm.Open(dialector, config)
	if err != nil {
		panic("failed to connect models")
	}
	DB = db

	// Configure connection pool for better concurrency handling
	sqlDB, err := db.DB()
	if err != nil {
		panic("failed to get database instance")
	}
	switch os.Getenv("DB_ADAPTER") {
	case "mysql", "postgres":
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
		sqlDB.SetConnMaxLifetime(time.Hour)
	default:
		// SQLite: WAL + busy_timeout + a modest pool.
		// Pragmas are set here (in addition to the DSN) to guarantee they
		// are applied to every connection the pool creates.
		if _, err := sqlDB.Exec(`PRAGMA journal_mode=WAL`); err != nil {
			log.Warnf("[DB] Could not set WAL mode: %v", err)
		}
		if _, err := sqlDB.Exec(`PRAGMA busy_timeout=10000`); err != nil {
			log.Warnf("[DB] Could not set busy_timeout: %v", err)
		}
		if _, err := sqlDB.Exec(`PRAGMA synchronous=NORMAL`); err != nil {
			log.Warnf("[DB] Could not set synchronous: %v", err)
		}
		// Keep the pool small — more connections do not help SQLite and
		// only increase lock contention. For true multi-user workloads
		// switch to PostgreSQL (DB_ADAPTER=postgres).
		sqlDB.SetMaxIdleConns(2)
		sqlDB.SetMaxOpenConns(8)
		sqlDB.SetConnMaxLifetime(time.Hour)
	}

	migrate()
}

// BeginTx starts a new transaction with default isolation level
// All database operations for multi-step processes should use this
func BeginTx() *gorm.DB {
	return DB.Begin(&sql.TxOptions{Isolation: sql.LevelReadCommitted})
}

func migrate() {
	// Migrate the schema in correct order (parent tables first)
	if err := DB.AutoMigrate(&User{}); err != nil {
		panic(fmt.Sprintf("[Migrate] Error user: %s", err))
	}
	if err := DB.AutoMigrate(&Channel{}); err != nil {
		panic(fmt.Sprintf("[Migrate] Error Channel: %s", err))
	}
	if err := DB.AutoMigrate(&Recording{}); err != nil {
		panic(fmt.Sprintf("[Migrate] Error Info: %s", err))
	}
	if err := DB.AutoMigrate(&Job{}); err != nil {
		panic(fmt.Sprintf("[Migrate] Error Job: %s", err))
	}
	if err := DB.AutoMigrate(&VideoPreview{}); err != nil {
		panic(fmt.Sprintf("[Migrate] Error VideoPreview: %s", err))
	}
	if err := DB.AutoMigrate(&VideoAnalysisResult{}); err != nil {
		panic(fmt.Sprintf("[Migrate] Error VideoAnalysisResult: %s", err))
	}
	if err := DB.AutoMigrate(&Setting{}); err != nil {
		panic(fmt.Sprintf("[Migrate] Error Setting: %s", err))
	}

	// Remove deprecated preview columns from recordings table
	if DB.Migrator().HasColumn(&Recording{}, "preview_stripe") {
		if err := DB.Migrator().DropColumn(&Recording{}, "preview_stripe"); err != nil {
			log.Warnf("[Migrate] Error dropping preview_stripe column: %s", err)
		} else {
			log.Infof("[Migrate] Dropped deprecated preview_stripe column")
		}
	}
	if DB.Migrator().HasColumn(&Recording{}, "preview_video") {
		if err := DB.Migrator().DropColumn(&Recording{}, "preview_video"); err != nil {
			log.Warnf("[Migrate] Error dropping preview_video column: %s", err)
		} else {
			log.Infof("[Migrate] Dropped deprecated preview_video column")
		}
	}
	if DB.Migrator().HasColumn(&Recording{}, "preview_cover") {
		if err := DB.Migrator().DropColumn(&Recording{}, "preview_cover"); err != nil {
			log.Warnf("[Migrate] Error dropping preview_cover column: %s", err)
		} else {
			log.Infof("[Migrate] Dropped deprecated preview_cover column")
		}
	}

	if err := InitSettings(); err != nil {
		log.Panicf("[Setting] Init error: %s", err)
	}

	// Drop frame_vectors if it has an older schema. Frame vectors are
	// derived/recomputable data so rebuilding them is safe.
	if sqlDB, err := DB.DB(); err == nil {
		var tableExists int
		sqlDB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name='frame_vectors'`).Scan(&tableExists)
		if tableExists > 0 {
			needsRebuild := false

			// Older iterations used vec0 auxiliary columns (+recording_id, +frame_index),
			// which cannot be used in KNN WHERE constraints.
			var tableSQL sql.NullString
			if err := sqlDB.QueryRow(`SELECT sql FROM sqlite_master WHERE name='frame_vectors'`).Scan(&tableSQL); err == nil && tableSQL.Valid {
				s := strings.ToLower(tableSQL.String)
				if strings.Contains(s, "+recording_id") || strings.Contains(s, "+frame_index") || strings.Contains(s, "+frame_timestamp") {
					needsRebuild = true
				}
			}

			// Ensure frame_index exists for consecutive similarity queries.
			if !needsRebuild {
				if _, colErr := sqlDB.Exec(`SELECT frame_index FROM frame_vectors LIMIT 0`); colErr != nil {
					needsRebuild = true
				}
			}

			if needsRebuild {
				if _, dropErr := sqlDB.Exec(`DROP TABLE IF EXISTS frame_vectors`); dropErr != nil {
					log.Warnf("[Migrate] Could not drop frame_vectors for schema update: %v", dropErr)
				} else {
					log.Infof("[Migrate] Dropped frame_vectors for schema update (will be rebuilt on next analysis)")
				}
			}
		}
	}
}
