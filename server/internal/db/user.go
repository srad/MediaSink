package db

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	UserID   uint   `json:"userId" gorm:"autoIncrement;primaryKey;column:user_id" extensions:"!x-nullable"`
	Username string `json:"username" gorm:"unique"`
	// Password holds the bcrypt hash and must never be serialized to a client.
	Password  string `json:"-"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// UserRepo is the user aggregate. Obtained from Store.Users(), so it always carries
// the handle its store was built with and cannot reach a different database.
type UserRepo struct {
	gorm *gorm.DB
}

// Exists reports whether the username is taken.
//
// The previous form returned an error to mean "taken", which left the caller unable to
// tell a duplicate name from an unreachable database — that is why a duplicate signup
// answers 500 rather than 409 today. This is the half of that defect the store owns;
// the status code is a wire change and belongs to phase 6. Until then the caller
// re-raises the same message, and internal/api/testdata/public_auth.golden pins it.
func (r UserRepo) Exists(username string) (bool, error) {
	var count int64
	if err := r.gorm.Model(&User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r UserRepo) Create(user *User) error {
	return r.gorm.Create(user).Error
}

func (r UserRepo) ByUsername(username string) (*User, error) {
	var user *User
	if err := r.gorm.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

func (r UserRepo) ByID(id uint) (*User, error) {
	var user *User
	if err := r.gorm.Where("user_id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}

	return user, nil
}
