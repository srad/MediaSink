package db

import (
	"path/filepath"
	"strings"
	"testing"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	"github.com/srad/mediasink/server/config"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Init panicked on every failure path, so none of them could be asserted on. These
// pin the errors Open returns instead.

func TestOpen_UnwritablePathReturnsAnError(t *testing.T) {
	// A directory component that does not exist. SQLite will not create it, so the
	// connection fails at open time rather than at first query.
	cfg := config.Cfg{DbFileName: filepath.Join(t.TempDir(), "no", "such", "dir", "x.db")}

	store, err := Open(t.Context(), cfg)
	if err == nil {
		t.Fatal("Open() on an unwritable path returned no error; it used to panic, it must not now succeed")
	}
	if store != nil {
		t.Errorf("Open() returned a non-nil store alongside an error: %#v", store)
	}
	if !strings.Contains(err.Error(), "connect database") {
		t.Errorf("error = %q, want it to say which stage failed", err)
	}
}

func TestOpen_UnreachableServerReturnsAnError(t *testing.T) {
	// Port 1 is reserved and nothing listens on it, so the driver's connect fails
	// without depending on whether a real Postgres happens to be running.
	cfg := config.Cfg{
		DBAdapter:  "postgres",
		DBHost:     "127.0.0.1",
		DBPort:     "1",
		DBUser:     "nobody",
		DBPassword: "nothing",
		DBName:     "nowhere",
	}

	store, err := Open(t.Context(), cfg)
	if err == nil {
		t.Fatal("Open() against an unreachable server returned no error")
	}
	if store != nil {
		t.Errorf("Open() returned a non-nil store alongside an error: %#v", store)
	}
	// The adapter is named so a misconfigured DB_ADAPTER is diagnosable from the
	// message alone.
	if !strings.Contains(err.Error(), `postgres`) {
		t.Errorf("error = %q, want it to name the adapter", err)
	}
}

func TestOpen_GoodPathMigratesEverySchema(t *testing.T) {
	cfg := config.Cfg{DbFileName: filepath.Join(t.TempDir(), "good.db")}

	store, err := Open(t.Context(), cfg)
	if err != nil {
		t.Fatalf("Open() on a writable path: %v", err)
	}
	if store == nil {
		t.Fatal("Open() returned a nil store and a nil error")
	}

	for _, target := range migrationTargets() {
		if !store.gorm.Migrator().HasTable(target.model) {
			t.Errorf("Migrate did not create the table for %s", target.name)
		}
	}
}

// Interim behaviour, pinned deliberately: the unconverted functions in this package
// still read the package-level DB, so Open has to keep assigning it. When the last of
// them moves onto a repository this test is deleted along with the global — that is
// the signal, not an accident.
func TestOpen_StillAssignsTheGlobalHandle(t *testing.T) {
	DB = nil

	store, err := Open(t.Context(), config.Cfg{DbFileName: filepath.Join(t.TempDir(), "global.db")})
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	if DB == nil {
		t.Fatal("Open() did not assign DB; every unconverted function in this package would nil-panic")
	}
	if DB != store.gorm {
		t.Error("the global handle and the store's handle are different connections")
	}
}

// An unknown adapter is not an error - it falls through to SQLite. That is load-
// bearing: DB_ADAPTER is empty in the default configuration and in the test
// harnesses, and both rely on getting SQLite.
func TestDialectorFor(t *testing.T) {
	tests := []struct {
		name    string
		adapter string
		want    string
	}{
		{"mysql", "mysql", "*mysql.Dialector"},
		{"postgres", "postgres", "*postgres.Dialector"},
		{"empty falls through to sqlite", "", "*sqlite.Dialector"},
		{"unknown falls through to sqlite", "cassandra", "*sqlite.Dialector"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := dialectorFor(config.Cfg{DBAdapter: test.adapter})
			var name string
			switch got.(type) {
			case *mysql.Dialector:
				name = "*mysql.Dialector"
			case *postgres.Dialector:
				name = "*postgres.Dialector"
			case *sqlite.Dialector:
				name = "*sqlite.Dialector"
			default:
				name = "unknown"
			}
			if name != test.want {
				t.Errorf("dialectorFor(%q) = %s, want %s", test.adapter, name, test.want)
			}
		})
	}
}

// migrate() panicked with the table name in the message. Migrate must keep naming it,
// because "AutoMigrate failed" on its own does not say which of the seven broke.
func TestMigrate_ErrorNamesTheFailingTable(t *testing.T) {
	sqlite_vec.Auto()

	handle, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "blocked.db")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open fixture database: %v", err)
	}

	// A view occupies the name `users`. gorm's HasTable only counts type='table', so
	// AutoMigrate tries CREATE TABLE and SQLite refuses: the name is taken.
	if err := handle.Exec(`CREATE VIEW users AS SELECT 1 AS user_id`).Error; err != nil {
		t.Fatalf("create blocking view: %v", err)
	}

	err = NewHandleFrom(handle).Migrate(t.Context())
	if err == nil {
		t.Fatal("Migrate() succeeded even though the users table could not be created")
	}
	if !strings.Contains(err.Error(), "User") {
		t.Errorf("error = %q, want it to name the User migration", err)
	}
}

// The three deprecated preview columns used to be dropped by three copy-pasted
// blocks; they are now a loop over deprecatedRecordingColumns. Nothing else would
// catch a column dropped from that slice - the migration would simply stop removing
// it and every test would still pass.
func TestMigrate_DropsEveryDeprecatedPreviewColumn(t *testing.T) {
	store, err := Open(t.Context(), config.Cfg{DbFileName: filepath.Join(t.TempDir(), "columns.db")})
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}

	// Put the old columns back, as an install predating the video_previews table
	// would have them.
	//
	// The backticks are load-bearing, and finding that out is worth recording. The
	// SQLite driver drops a column by rewriting the stored CREATE TABLE text, and
	// ddlmod.go:274 removeColumn only matches a name that is quoted or space-
	// delimited. migrator.go:163 throws away the bool it returns, so an unquoted
	// column is recreated intact while DropColumn still reports success. Real
	// installs got these columns from AutoMigrate, which quotes them, so the
	// production path works — but a fixture that adds them unquoted tests nothing.
	for _, column := range deprecatedRecordingColumns {
		if err := store.gorm.Exec("ALTER TABLE recordings ADD COLUMN `" + column + "` text").Error; err != nil {
			t.Fatalf("re-adding %s: %v", column, err)
		}
		if !store.gorm.Migrator().HasColumn(&Recording{}, column) {
			t.Fatalf("precondition failed: %s was not re-added", column)
		}
	}

	// Migrating again must remove them, and must still succeed on a schema that is
	// already up to date.
	if err := store.Migrate(t.Context()); err != nil {
		t.Fatalf("second Migrate(): %v", err)
	}

	for _, column := range deprecatedRecordingColumns {
		if store.gorm.Migrator().HasColumn(&Recording{}, column) {
			t.Errorf("deprecated column %s survived Migrate()", column)
		}
	}
}

// Guards the list itself: dropping an entry is the silent failure above, so pin what
// the three names are.
func TestDeprecatedRecordingColumns(t *testing.T) {
	want := []string{"preview_stripe", "preview_video", "preview_cover"}
	if len(deprecatedRecordingColumns) != len(want) {
		t.Fatalf("deprecatedRecordingColumns = %v, want %v", deprecatedRecordingColumns, want)
	}
	for i, name := range want {
		if deprecatedRecordingColumns[i] != name {
			t.Errorf("deprecatedRecordingColumns[%d] = %q, want %q", i, deprecatedRecordingColumns[i], name)
		}
	}
}

// Close releases the pool. Until it existed, app.Shutdown stopped the workers and the
// HTTP server but left the database connection open for the life of the process.
func TestHandleClose(t *testing.T) {
	handle, err := Open(t.Context(), config.Cfg{DbFileName: filepath.Join(t.TempDir(), "close.db")})
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}

	// Works before the close.
	if err := handle.Gorm().Exec(`SELECT 1`).Error; err != nil {
		t.Fatalf("query before Close: %v", err)
	}

	if err := handle.Close(); err != nil {
		t.Fatalf("Close(): %v", err)
	}

	// And fails after it, which is what proves the pool actually shut rather than
	// Close silently doing nothing.
	if err := handle.Gorm().Exec(`SELECT 1`).Error; err == nil {
		t.Error("a query succeeded after Close(); the connection pool is still open")
	}
}

// These two cover the free BeginTx, which is what production actually calls -
// recording.go:251,284,327 and channel_id.go:121,150. They used to exercise
// Store.BeginTx, which had no callers outside these tests and has been deleted.
// Open assigns the package global, so the free form reaches this database.
func TestBeginTx_RollbackDiscardsTheWrite(t *testing.T) {
	store, err := Open(t.Context(), config.Cfg{DbFileName: filepath.Join(t.TempDir(), "tx.db")})
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}

	tx := BeginTx()
	if tx.Error != nil {
		t.Fatalf("BeginTx(): %v", tx.Error)
	}
	if err := tx.Create(&Setting{SettingKey: "rolled_back", SettingValue: "1", SettingType: "int"}).Error; err != nil {
		t.Fatalf("insert inside the transaction: %v", err)
	}
	if err := tx.Rollback().Error; err != nil {
		t.Fatalf("Rollback(): %v", err)
	}

	var count int64
	if err := store.gorm.Model(&Setting{}).Where("setting_key = ?", "rolled_back").Count(&count).Error; err != nil {
		t.Fatalf("count after rollback: %v", err)
	}
	if count != 0 {
		t.Errorf("row count after rollback = %d, want 0: BeginTx did not open a real transaction", count)
	}
}

func TestBeginTx_CommitPersistsTheWrite(t *testing.T) {
	store, err := Open(t.Context(), config.Cfg{DbFileName: filepath.Join(t.TempDir(), "tx.db")})
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}

	tx := BeginTx()
	if tx.Error != nil {
		t.Fatalf("BeginTx(): %v", tx.Error)
	}
	if err := tx.Create(&Setting{SettingKey: "committed", SettingValue: "1", SettingType: "int"}).Error; err != nil {
		t.Fatalf("insert inside the transaction: %v", err)
	}
	if err := tx.Commit().Error; err != nil {
		t.Fatalf("Commit(): %v", err)
	}

	var count int64
	if err := store.gorm.Model(&Setting{}).Where("setting_key = ?", "committed").Count(&count).Error; err != nil {
		t.Fatalf("count after commit: %v", err)
	}
	if count != 1 {
		t.Errorf("row count after commit = %d, want 1", count)
	}
}

// configurePool was lifted verbatim out of Init. The pool sizes differ by an order of
// magnitude between SQLite and the server drivers, and picking the wrong branch would
// not fail anything visibly - SQLite would just contend under load.
func TestConfigurePool(t *testing.T) {
	tests := []struct {
		name        string
		adapter     string
		wantMaxOpen int
	}{
		{"mysql gets the server-sized pool", "mysql", 100},
		{"postgres gets the server-sized pool", "postgres", 100},
		{"sqlite is kept small", "", 8},
		{"an unknown adapter is treated as sqlite", "cassandra", 8},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Any real *sql.DB will do: configurePool only sets pool limits, and the
			// pragma statements it issues on the SQLite branch are best-effort.
			handle, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "pool.db")), &gorm.Config{
				Logger: logger.Default.LogMode(logger.Silent),
			})
			if err != nil {
				t.Fatalf("open fixture database: %v", err)
			}
			sqlDB, err := handle.DB()
			if err != nil {
				t.Fatalf("sql.DB handle: %v", err)
			}

			configurePool(config.Cfg{DBAdapter: test.adapter}, sqlDB)

			if got := sqlDB.Stats().MaxOpenConnections; got != test.wantMaxOpen {
				t.Errorf("MaxOpenConnections = %d, want %d", got, test.wantMaxOpen)
			}
		})
	}
}

// Migrate seeds settings and reads the embedding model. Both used to go through the
// package-level handle, which is invisible while Open is the only caller — it assigns
// that global to the same connection. Migrating a separately opened connection makes it
// visible: settings would be written into the wrong database.
func TestMigrate_WritesToItsOwnHandleNotTheGlobal(t *testing.T) {
	// The global points at a database that must stay untouched.
	other, err := Open(t.Context(), config.Cfg{DbFileName: filepath.Join(t.TempDir(), "other.db")})
	if err != nil {
		t.Fatalf("open the decoy database: %v", err)
	}
	if err := other.gorm.Where("1 = 1").Delete(&Setting{}).Error; err != nil {
		t.Fatalf("clear the decoy settings: %v", err)
	}

	sqlite_vec.Auto()
	target, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "target.db")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open the target database: %v", err)
	}

	if err := NewHandleFrom(target).Migrate(t.Context()); err != nil {
		t.Fatalf("Migrate() on the target: %v", err)
	}

	var seededInTarget int64
	if err := target.Model(&Setting{}).Where("setting_key = ?", ReqInterval).Count(&seededInTarget).Error; err != nil {
		t.Fatalf("count settings in the target: %v", err)
	}
	if seededInTarget != 1 {
		t.Errorf("target settings rows = %d, want 1: Migrate seeded somewhere else", seededInTarget)
	}

	var leakedIntoGlobal int64
	if err := other.gorm.Model(&Setting{}).Where("setting_key = ?", ReqInterval).Count(&leakedIntoGlobal).Error; err != nil {
		t.Fatalf("count settings in the decoy: %v", err)
	}
	if leakedIntoGlobal != 0 {
		t.Errorf("decoy settings rows = %d, want 0: Migrate wrote through the package global", leakedIntoGlobal)
	}
}

// Open must not hand back a half-migrated store when Migrate fails.
func TestOpen_PropagatesMigrationFailure(t *testing.T) {
	sqlite_vec.Auto()
	path := filepath.Join(t.TempDir(), "blocked.db")

	handle, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open fixture database: %v", err)
	}
	if err := handle.Exec(`CREATE VIEW users AS SELECT 1 AS user_id`).Error; err != nil {
		t.Fatalf("create blocking view: %v", err)
	}
	sqlDB, err := handle.DB()
	if err != nil {
		t.Fatalf("sql.DB handle: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close fixture handle: %v", err)
	}

	store, err := Open(t.Context(), config.Cfg{DbFileName: path})
	if err == nil {
		t.Fatal("Open() returned no error although Migrate could not create users")
	}
	if store != nil {
		t.Errorf("Open() returned a store despite a failed migration: %#v", store)
	}
}
