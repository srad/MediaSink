package db

import (
	"database/sql"
	"fmt"

	"github.com/srad/mediasink/server/config"
	"gorm.io/gorm"
)

// DB is the handle the not-yet-converted functions in this package still read.
// Store.Open assigns it. It goes away with the last of them; see ROADMAP.md, phase 2b.
var DB *gorm.DB

// sqlDSN builds the connection string for the server-based drivers. MySQL and
// Postgres share it verbatim, which is why it lives here rather than being spelled
// out twice inside dialectorFor.
func sqlDSN(cfg config.Cfg) string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Europe/Berlin",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort)
}

// BeginTx starts a new transaction with default isolation level
// All database operations for multi-step processes should use this
//
// Superseded by Store.BeginTx. This form survives only for the free functions in
// this package that have not moved onto a repository yet.
func BeginTx() *gorm.DB {
	return DB.Begin(&sql.TxOptions{Isolation: sql.LevelReadCommitted})
}
