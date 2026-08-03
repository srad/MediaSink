package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/server/config"
	"github.com/srad/mediasink/server/internal/analysis/detectors/onnx"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Store owns the database handle. It replaces the package-level `DB` global: every
// consumer is handed a *Store rather than reaching for package state.
//
// Consumers reach an aggregate through a per-aggregate repository rather than through
// methods hung directly off Store, which would otherwise grow to roughly 95 of them.
// Users and Settings landed in phase 2b.2; Channels, Recordings, Jobs and Vectors
// follow as each aggregate migrates off the global (see ROADMAP.md, phase 2b). Until
// the last of them, `Open` still assigns `DB` so the unconverted functions in this
// package keep working.
type Store struct {
	gorm *gorm.DB
}

// Users is the user aggregate. See UserRepo.
func (s *Store) Users() UserRepo { return UserRepo{gorm: s.gorm} }

// Settings is the settings aggregate. See SettingRepo.
func (s *Store) Settings() SettingRepo { return SettingRepo{gorm: s.gorm} }

// Open connects, configures the pool and migrates. It replaces Init, which panicked
// on every failure path and so could not be exercised by a test.
func Open(cfg config.Cfg) (*Store, error) {
	gormLogger := logger.New(
		log.New(),
		logger.Config{
			//SlowThreshold:             time.Second,  // Slow SQL threshold
			LogLevel:                  logger.Warn, // Log level
			IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
			//ParameterizedQueries:      true,         // Don't include params in the SQL log
			Colorful: true, // Disable color
		},
	)

	// Named gormCfg, not config: a local named `config` would shadow the imported
	// config package for the rest of this function.
	gormCfg := &gorm.Config{
		Logger:                                   gormLogger,
		DisableForeignKeyConstraintWhenMigrating: false, // Enable foreign key constraints for data integrity
	}

	handle, err := gorm.Open(dialectorFor(cfg), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("connect database (adapter %q): %w", cfg.DBAdapter, err)
	}

	sqlDB, err := handle.DB()
	if err != nil {
		return nil, fmt.Errorf("obtain sql.DB handle: %w", err)
	}
	configurePool(cfg, sqlDB)

	store := &Store{gorm: handle}

	// Interim, removed with the last of the unconverted functions in this package.
	DB = handle

	if err := store.Migrate(); err != nil {
		return nil, err
	}

	return store, nil
}

// NewStoreFrom wraps a handle that is already open. Its purpose is test fixtures,
// which build their own in-memory database and then need a Store over it; production
// code goes through Open so that pool configuration and migration are not skipped.
func NewStoreFrom(handle *gorm.DB) *Store {
	return &Store{gorm: handle}
}

// dialectorFor picks the driver. Split out of Open so the choice is testable without
// opening a connection.
func dialectorFor(cfg config.Cfg) gorm.Dialector {
	switch cfg.DBAdapter {
	case "mysql":
		return mysql.New(mysql.Config{DSN: sqlDSN(cfg)})
	case "postgres":
		return postgres.New(postgres.Config{DSN: sqlDSN(cfg)})
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
		return sqlite.Open(dsn)
	}
}

// configurePool sizes the connection pool for the adapter in use. The pragma failures
// are warnings, not errors: they are already set on the DSN, and this is a belt-and-
// braces pass to cover every connection the pool later creates.
func configurePool(cfg config.Cfg, sqlDB *sql.DB) {
	switch cfg.DBAdapter {
	case "mysql", "postgres":
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
		sqlDB.SetConnMaxLifetime(time.Hour)
	default:
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
}

// BeginTx starts a new transaction with default isolation level.
// All database operations for multi-step processes should use this.
func (s *Store) BeginTx() *gorm.DB {
	return s.gorm.Begin(&sql.TxOptions{Isolation: sql.LevelReadCommitted})
}

// migrationTargets is the schema in dependency order: parent tables first, so the
// foreign key constraints AutoMigrate emits always have something to point at.
func migrationTargets() []struct {
	name  string
	model any
} {
	return []struct {
		name  string
		model any
	}{
		{"User", &User{}},
		{"Channel", &Channel{}},
		{"Recording", &Recording{}},
		{"Job", &Job{}},
		{"VideoPreview", &VideoPreview{}},
		{"VideoAnalysisResult", &VideoAnalysisResult{}},
		{"Setting", &Setting{}},
	}
}

// deprecatedRecordingColumns were replaced by the video_previews table. Dropping them
// is best-effort: an install that never had them is not an error.
var deprecatedRecordingColumns = []string{"preview_stripe", "preview_video", "preview_cover"}

// Migrate brings the schema up to date. Where the old migrate() panicked, this
// returns.
//
// It stops at the first AutoMigrate failure rather than collecting every error:
// the targets are ordered parent-first, so once Channel fails, Recording and Job
// fail too for a reason that is not their own, and joining those would bury the
// one error that matters under cascade noise.
func (s *Store) Migrate() error {
	for _, target := range migrationTargets() {
		if err := s.gorm.AutoMigrate(target.model); err != nil {
			return fmt.Errorf("migrate %s: %w", target.name, err)
		}
	}

	for _, column := range deprecatedRecordingColumns {
		if !s.gorm.Migrator().HasColumn(&Recording{}, column) {
			continue
		}
		if err := s.gorm.Migrator().DropColumn(&Recording{}, column); err != nil {
			log.Warnf("[Migrate] Error dropping %s column: %s", column, err)
		} else {
			log.Infof("[Migrate] Dropped deprecated %s column", column)
		}
	}

	if err := s.Settings().init(); err != nil {
		return fmt.Errorf("initialise settings: %w", err)
	}

	s.dropStaleFrameVectors()

	return nil
}

// dropStaleFrameVectors drops frame_vectors when it no longer matches what the
// current code produces — either an older table schema or vectors from a
// different embedding model. Frame vectors are derived/recomputable data, so
// rebuilding them is safe; the startup backfill re-queues the analysis.
//
// Every failure here is a warning rather than an error: being unable to tell whether
// the table is stale is not a reason to refuse to boot.
func (s *Store) dropStaleFrameVectors() {
	sqlDB, err := s.gorm.DB()
	if err != nil {
		log.Warnf("[dropStaleFrameVectors] Could not obtain the sql.DB handle: %s", err)
		return
	}

	var tableExists int
	// If this fails we cannot tell whether the table is there; skipping is safer
	// than assuming it is absent and silently leaving stale vectors in place.
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name='frame_vectors'`).Scan(&tableExists); err != nil {
		log.Warnf("[dropStaleFrameVectors] Could not determine whether frame_vectors exists: %s", err)
		return
	}
	if tableExists == 0 {
		return
	}

	needsRebuild := false

	// Older iterations used vec0 auxiliary columns (+recording_id, +frame_index),
	// which cannot be used in KNN WHERE constraints.
	var tableSQL sql.NullString
	if err := sqlDB.QueryRow(`SELECT sql FROM sqlite_master WHERE name='frame_vectors'`).Scan(&tableSQL); err == nil && tableSQL.Valid {
		schema := strings.ToLower(tableSQL.String)
		if strings.Contains(schema, "+recording_id") || strings.Contains(schema, "+frame_index") || strings.Contains(schema, "+frame_timestamp") {
			needsRebuild = true
		}
	}

	// Ensure frame_index exists for consecutive similarity queries.
	if !needsRebuild {
		if _, colErr := sqlDB.Exec(`SELECT frame_index FROM frame_vectors LIMIT 0`); colErr != nil {
			needsRebuild = true
		}
	}

	// The embedding model is part of the contract too. Vectors from a
	// different model have the same dimension but occupy a different
	// space, so the table would not error — old and new rows would just
	// be silently compared against each other and rank as noise.
	reason := "schema update"
	if !needsRebuild {
		storedModel, modelErr := s.Settings().EmbeddingModel()
		if modelErr != nil {
			log.Warnf("[Migrate] Could not read the stored embedding model: %v", modelErr)
		} else if storedModel != onnx.DefaultModelName {
			log.Infof("[Migrate] Embedding model changed (%s -> %s), stored frame vectors are no longer comparable",
				storedModel, onnx.DefaultModelName)
			needsRebuild = true
			reason = "embedding model change"
		}
	}

	if !needsRebuild {
		return
	}

	if _, dropErr := sqlDB.Exec(`DROP TABLE IF EXISTS frame_vectors`); dropErr != nil {
		log.Warnf("[Migrate] Could not drop frame_vectors for %s: %v", reason, dropErr)
	} else {
		log.Infof("[Migrate] Dropped frame_vectors for %s (will be rebuilt on next analysis)", reason)
	}
}
