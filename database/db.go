package database

import (
	"database/sql"
	"fmt"
	"os"
	"time"

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
		// SQLite3
		dialector = sqlite.Open(cfg.DbFileName)
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
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

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
}
