package db

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/srad/mediasink/server/config"
	"gorm.io/gorm"
)

// userStore opens a throwaway database through the real Open, so the repo runs against
// the schema Migrate actually produces rather than a hand-rolled AutoMigrate. A temp
// file rather than ":memory:", for the reason recorded in store_test.go: every
// ":memory:" connection is its own private database.
func userStore(t *testing.T) *Store {
	t.Helper()

	store, err := Open(config.Cfg{DbFileName: filepath.Join(t.TempDir(), "users.db")})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return store
}

func TestUserRepoExists(t *testing.T) {
	users := userStore(t).Users()

	taken, err := users.Exists("nobody@example.com")
	if err != nil {
		t.Fatalf("Exists on an empty table: %v", err)
	}
	if taken {
		t.Error("Exists reported a username as taken on an empty table")
	}

	if err := users.Create(&User{Username: "someone@example.com", Password: "hash"}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if taken, err = users.Exists("someone@example.com"); err != nil {
		t.Fatalf("Exists after Create: %v", err)
	}
	if !taken {
		t.Error("Exists reported a seeded username as free")
	}

	// The distinction the old error-as-boolean signature could not express: a free
	// username is (false, nil), never an error.
	if taken, err = users.Exists("still-nobody@example.com"); err != nil || taken {
		t.Errorf("Exists(free username) = (%v, %v), want (false, <nil>)", taken, err)
	}
}

// Username carries gorm:"unique", so the schema is the backstop if two signups race
// past Exists. Nothing else in the tree asserts that Migrate actually emits the
// constraint.
func TestUserRepoCreateRejectsADuplicateUsername(t *testing.T) {
	users := userStore(t).Users()

	if err := users.Create(&User{Username: "dupe@example.com", Password: "hash"}); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	if err := users.Create(&User{Username: "dupe@example.com", Password: "other-hash"}); err == nil {
		t.Error("Create accepted a duplicate username; the unique constraint is missing")
	}
}

func TestUserRepoByUsername(t *testing.T) {
	users := userStore(t).Users()

	seeded := &User{Username: "found@example.com", Password: "hash"}
	if err := users.Create(seeded); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := users.ByUsername("found@example.com")
	if err != nil {
		t.Fatalf("ByUsername: %v", err)
	}
	if got.UserID != seeded.UserID || got.Password != "hash" {
		t.Errorf("ByUsername = %+v, want the seeded user (id %d)", got, seeded.UserID)
	}

	if _, err = users.ByUsername("absent@example.com"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("ByUsername(absent) error = %v, want gorm.ErrRecordNotFound", err)
	}
}

func TestUserRepoByID(t *testing.T) {
	users := userStore(t).Users()

	seeded := &User{Username: "byid@example.com", Password: "hash"}
	if err := users.Create(seeded); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := users.ByID(seeded.UserID)
	if err != nil {
		t.Fatalf("ByID: %v", err)
	}
	if got.Username != "byid@example.com" {
		t.Errorf("ByID = %+v, want %q", got, "byid@example.com")
	}

	if _, err = users.ByID(seeded.UserID + 1000); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("ByID(absent) error = %v, want gorm.ErrRecordNotFound", err)
	}
}

// The repo must read and write the handle its own store was built with. The package
// global is still assigned by Open until phase 2b.6, so a repo that reached for it
// instead would pass every test above and silently use the wrong database — which is
// exactly the defect 2b.1 found in Migrate.
func TestUserRepoWritesToItsOwnHandleNotTheGlobal(t *testing.T) {
	first := userStore(t)
	second := userStore(t) // Open reassigns the global DB to this one.

	if err := first.Users().Create(&User{Username: "scoped@example.com", Password: "hash"}); err != nil {
		t.Fatalf("Create against the first store: %v", err)
	}

	taken, err := second.Users().Exists("scoped@example.com")
	if err != nil {
		t.Fatalf("Exists against the second store: %v", err)
	}
	if taken {
		t.Error("a write through the first store's repo was visible from the second store")
	}
}
